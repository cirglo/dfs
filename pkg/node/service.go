package node

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Service interface {
	GetBlockIds() ([]string, error)
	GetBlocks() ([]BlockInfo, error)
	WriteBlock(blockInfo BlockInfo, data []byte) error
	DeleteBlock(id string) error
	ReadBlock(id string) ([]byte, BlockInfo, error)
}

type BlockInfo struct {
	ID       string `json:"id"`
	Sequence uint64 `json:"sequence"`
	Length   uint32 `json:"length"`
	Path     string `json:"path"`
	CRC      uint32 `json:"crc"`
}

type ServiceOpts struct {
	Logger              *logrus.Logger
	ID                  string
	Location            string
	Dir                 fs.FileInfo
	HealthCheckInterval time.Duration
	ValidateCRCInterval time.Duration
}

func (o *ServiceOpts) Validate() error {
	if o.Logger == nil {
		return fmt.Errorf("logger is required")
	}

	if len(o.ID) == 0 {
		return fmt.Errorf("id is required")
	}

	if len(o.Location) == 0 {
		return fmt.Errorf("location is required")
	}

	if o.Dir == nil {
		return fmt.Errorf("dir is required")
	}

	if !o.Dir.IsDir() {
		return fmt.Errorf("dir is not a directory: %s", o.Dir)
	}

	return nil
}

type service struct {
	opts       ServiceOpts
	logEntry   *logrus.Entry
	lock       sync.RWMutex
	blockInfos map[string]BlockInfo
}

func NewService(opts ServiceOpts) (Service, error) {
	err := opts.Validate()
	if err != nil {
		return nil, fmt.Errorf("options are not valid: %w", err)
	}

	s := service{
		opts: opts,
		logEntry: logrus.WithFields(logrus.Fields{
			"id":       opts.ID,
			"location": opts.Location,
		})}

	s.logEntry.Info("Initializing")
	err = s.healthCheck()
	if err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}

	err = s.validateCRC()
	if err != nil {
		return nil, fmt.Errorf("validate CRC failed: %w", err)
	}

	go func() {
		ticker := time.NewTicker(opts.HealthCheckInterval)
		for _ = range ticker.C {
			err := s.healthCheck()
			if err != nil {
				s.logEntry.WithError(err).Fatal("health check failed")
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(opts.ValidateCRCInterval)
		for _ = range ticker.C {
			err := s.validateCRC()
			if err != nil {
				s.logEntry.WithError(err).Fatal("validate CRC failed")
			}
		}
	}()

	return &s, nil
}

func (s *service) GetBlockIds() ([]string, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	ids := make([]string, 0, len(s.blockInfos))

	for id, _ := range s.blockInfos {
		ids = append(ids, id)
	}

	return ids, nil
}

func (s *service) GetBlocks() ([]BlockInfo, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	blockInfos := make([]BlockInfo, 0, len(s.blockInfos))

	for _, blockInfo := range s.blockInfos {
		blockInfos = append(blockInfos, blockInfo)
	}

	return blockInfos, nil
}

func (s *service) WriteBlock(blockInfo BlockInfo, data []byte) error {
	if blockInfo.CRC != crc32.ChecksumIEEE(data) {
		return fmt.Errorf("invalid checksum (mismatch)")
	}
	if blockInfo.Length != uint32(len(data)) {
		return fmt.Errorf("invalid length")
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	err := s.writeBlockData(blockInfo.ID, data)
	if err != nil {
		return fmt.Errorf("failed to write data file: %w", err)
	}

	err = s.writeBlockInfo(blockInfo)
	if err != nil {
		return fmt.Errorf("failed to write meta data file: %w", err)
	}

	s.blockInfos[blockInfo.ID] = blockInfo

	return nil
}

func (s *service) DeleteBlock(id string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	mdFilePath := s.createMetaDataPath(id)
	dataFilePath := s.createDataPath(id)
	logEntry := s.logEntry.WithFields(logrus.Fields{
		"block-id":            id,
		"meta-data-file-path": mdFilePath,
		"data-file-path":      dataFilePath,
	})
	err := os.Remove(mdFilePath)
	if err != nil {
		logEntry.WithError(err).Error("failed to delete metadata file")
	}

	err = os.Remove(dataFilePath)
	if err != nil {
		logEntry.WithError(err).Error("failed to delete data file")
	}

	delete(s.blockInfos, id)

	return nil
}

func (s *service) ReadBlock(id string) ([]byte, BlockInfo, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	blockInfo, ok := s.blockInfos[id]
	if !ok {
		return nil, BlockInfo{}, fmt.Errorf("block id %s not found", id)
	}

	data, crc, err := s.readBlockData(id)
	if err != nil {
		return nil, BlockInfo{}, fmt.Errorf("could not read data for block id %s : %w", id, err)
	}

	if blockInfo.CRC != crc {
		return nil, BlockInfo{}, fmt.Errorf("invalid checksum (mismatch)")
	}

	if blockInfo.Length != uint32(len(data)) {
		return nil, blockInfo, fmt.Errorf("invalid length")
	}

	return data, blockInfo, nil
}

func CopyBlock(id string, source Service, dest Service) error {
	data, blockInfo, err := source.ReadBlock(id)
	if err != nil {
		return fmt.Errorf("failed to read data for block id %s : %w", id, err)
	}

	err = dest.WriteBlock(blockInfo, data)
	if err != nil {
		return fmt.Errorf("failed to write data for block id %s : %w", id, err)
	}

	return nil
}

func (s *service) healthCheck() error {
	toDelete := map[string]bool{}

	s.lock.RLock()
	for id, blockInfo := range s.blockInfos {
		mdFilePath := s.createMetaDataPath(id)
		dataFilePath := s.createDataPath(id)
		logEntry := s.logEntry.WithFields(logrus.Fields{
			"block-id":       id,
			"block-info":     blockInfo,
			"md-file-path":   mdFilePath,
			"data-file-path": dataFilePath,
		})

		_, err := os.Stat(mdFilePath)
		if err != nil {
			logEntry.WithError(err).Warn("failed to stat metadata file")
			toDelete[id] = true
		}

		_, err = os.Stat(dataFilePath)
		if err != nil {
			logEntry.WithError(err).Warn("failed to stat data file")
			toDelete[id] = true
		}
	}
	s.lock.RUnlock()

	if len(toDelete) > 0 {
		s.logEntry.Warn("deleting bad blocks")
		for id, _ := range toDelete {
			logEntry := s.logEntry.WithField("block-id", id)
			err := s.DeleteBlock(id)
			if err != nil {
				logEntry.WithError(err).Warn("failed to delete bad block")
			}
		}
		s.logEntry.Warn("finished deleting bad blocks")
	}

	return nil

}

func (s *service) validateCRC() error {
	toDelete := map[string]bool{}

	files, err := os.ReadDir(s.opts.Dir.Name())
	if err != nil {
		return fmt.Errorf("cannot read dir %s : %w", s.opts.Dir, err)
	}

	s.lock.RLock()
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		name := f.Name()

		if !strings.HasSuffix(name, ".data") {
			continue
		}

		logEntry := s.logEntry.WithField("file-path", f.Name())

		fi, err := f.Info()
		if err != nil {
			logEntry.WithError(err).Error("cannot read file info")
			continue
		}

		id := strings.TrimSuffix(name, ".data")

		logEntry = logEntry.WithField("block-id", id)
		_, crc, err := s.readBlockData(fi.Name())
		if err != nil {
			logEntry.WithError(err).Error("cannot read block data")
			toDelete[id] = true
			continue
		}
		logEntry = logEntry.WithField("crc", crc)

		bifi, err := os.Stat(fmt.Sprintf("%s.md.json"))
		if err != nil {
			logEntry.WithError(err).Error("cannot stat metadate file")
			toDelete[id] = true
			continue
		}

		blockInfo, err := s.readBlockInfo(bifi.Name())
		if err != nil {
			logEntry.WithError(err).Error("cannot read block info")
			toDelete[id] = true
			continue
		}

		logEntry = logEntry.WithField("block-info", blockInfo)

		if crc != blockInfo.CRC {
			logEntry.WithError(err).Error("crc mismatch")
			toDelete[id] = true
			continue
		}
	}
	s.lock.RUnlock()

	for id, _ := range toDelete {
		err := s.DeleteBlock(id)
		if err != nil {
			s.logEntry.WithError(err).WithField("block-id", id).Warn("failed to delete bad block")
		}
	}

	return nil
}

func (s *service) createMetaDataPath(id string) string {
	return s.createPath(id, "md.json")
}

func (s *service) createDataPath(id string) string {
	return s.createPath(id, "data")
}

func (s *service) createPath(id string, suffix string) string {
	return filepath.Join(s.opts.Dir.Name(), fmt.Sprintf("%s.%s", id, suffix))
}

func (s *service) readBlockInfo(id string) (BlockInfo, error) {
	bi, err := s.readBlockInfoFromPath(s.createMetaDataPath(id))
	if err != nil {
		return bi, fmt.Errorf("failed to read block info id %s: %w", id, err)
	}

	return bi, nil
}

func (s *service) readBlockInfoFromPath(path string) (BlockInfo, error) {
	bi := BlockInfo{}
	b, err := os.ReadFile(path)
	if err != nil {
		return bi, fmt.Errorf("cannot read block info metadata file %s: %w", path, err)
	}

	err = json.Unmarshal(b, &bi)
	if err != nil {
		return bi, fmt.Errorf("cannot parse block info metadata file %s: %w", path, err)
	}

	return bi, nil
}

func (s *service) writeBlockInfo(bi BlockInfo) error {
	err := s.writeBlockInfoToPath(s.createMetaDataPath(bi.ID), bi)
	if err != nil {
		return fmt.Errorf("cannot write block info metadata id %s: %w", bi.ID, err)
	}

	return nil
}

func (s *service) writeBlockInfoToPath(path string, bi BlockInfo) error {
	b, err := json.Marshal(bi)
	if err != nil {
		return fmt.Errorf("cannot marshal block info metadata file %s: %w", path, err)
	}

	err = os.WriteFile(path, b, 0600)
	if err != nil {
		return fmt.Errorf("cannot write block info metadata file %s: %w", path, err)
	}

	return nil
}

func (s *service) readBlockData(id string) ([]byte, uint32, error) {
	b, crc, err := s.readBlockDataFromPath(s.createDataPath(id))
	if err != nil {
		return nil, 0, fmt.Errorf("cannot read block info metadata id %s: %w", id, err)
	}

	return b, crc, nil
}

func (s *service) readBlockDataFromPath(path string) ([]byte, uint32, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return b, 0, fmt.Errorf("cannot read block data metadata file %s: %w", path, err)
	}
	crc := crc32.ChecksumIEEE(b)

	return b, crc, nil
}

func (s *service) writeBlockData(id string, data []byte) error {
	err := s.writeBlockDataToPath(s.createDataPath(id), data)
	if err != nil {
		return fmt.Errorf("cannot write block data metadata id %s: %w", id, err)
	}

	return nil
}

func (s *service) writeBlockDataToPath(path string, data []byte) error {
	err := os.WriteFile(path, data, 0600)
	if err != nil {
		return fmt.Errorf("cannot write block data metadata file %s: %w", path, err)
	}

	return nil
}
