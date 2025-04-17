package node

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/cirglo.com/dfs/pkg/proto"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
)

type BlockService interface {
	GetBlockIds() ([]string, error)
	GetBlocks() ([]BlockInfo, error)
	WriteBlock(id string, path string, sequence uint64, data []byte) error
	DeleteBlock(id string) error
	ReadBlock(id string) ([]byte, BlockInfo, error)
	Report() error
	HealthCheck() error
	ValidateCRC() error
}

type BlockInfo struct {
	ID           string `gorm:"primaryKey;uniqueIndex:idx_block_info;not null"`
	Sequence     uint64 `gorm:"not null;uniqueIndex:idx_block_info'"`
	Length       uint32 `gorm:"not null"`
	Path         string `gorm:"not null"`
	DataFilePath string `gorm:"not null"`
	CRC          uint32 `gorm:"not null"`
}

func (bi *BlockInfo) BeforeSave(_ *gorm.DB) error {
	bi.ID = strings.TrimSpace(bi.ID)
	bi.Path = strings.TrimSpace(bi.Path)
	bi.DataFilePath = strings.TrimSpace(bi.DataFilePath)

	if len(bi.ID) == 0 {
		return fmt.Errorf("block id is empty")
	}

	if len(bi.Path) == 0 {
		return fmt.Errorf("path is empty")
	}

	if len(bi.DataFilePath) == 0 {
		return fmt.Errorf("data file path is empty")
	}

	if bi.Length == 0 {
		return fmt.Errorf("length is zero")
	}

	return nil
}

type BlockServiceOpts struct {
	Logger             *logrus.Logger
	Host               string
	DB                 *gorm.DB
	Dir                string
	NotificationClient proto.NotificationClient
}

func (o *BlockServiceOpts) Validate() error {
	if o.Logger == nil {
		return fmt.Errorf("logger is required")
	}

	if o.DB == nil {
		return fmt.Errorf("db is required")
	}

	dirStat, err := os.Stat(o.Dir)
	if err != nil {
		return fmt.Errorf("could not stat dir %s: %w", o.Dir, err)
	}

	if !dirStat.IsDir() {
		return fmt.Errorf("dir is not a directory: %s", o.Dir)
	}

	if o.Host == "" {
		return fmt.Errorf("host is required")
	}

	if o.NotificationClient == nil {
		return fmt.Errorf("notificationClient is required")
	}

	return nil
}

type service struct {
	opts BlockServiceOpts
}

func NewBlockService(opts BlockServiceOpts) (BlockService, error) {
	err := opts.Validate()
	if err != nil {
		return nil, fmt.Errorf("options are not valid: %w", err)
	}

	opts.Logger.WithFields(logrus.Fields{
		"dir":  opts.Dir,
		"host": opts.Host,
	}).Info("Constructing new service")

	s := service{opts: opts}

	return &s, nil
}

func (s *service) GetBlockIds() ([]string, error) {
	var blockIds []string
	err := s.opts.DB.Transaction(func(tx *gorm.DB) error {
		blockInfos := []BlockInfo{}
		err := tx.Find(&blockInfos).Error
		if err != nil {
			return fmt.Errorf("failed to get block ids: %w", err)
		}

		for _, blockInfo := range blockInfos {
			blockIds = append(blockIds, blockInfo.ID)
		}

		return nil
	}, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get block ids: %w", err)
	}

	return blockIds, nil
}

func (s *service) GetBlocks() ([]BlockInfo, error) {
	var blockInfos []BlockInfo

	err := s.opts.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Find(&blockInfos).Error
		if err != nil {
			return fmt.Errorf("failed to get blocks: %w", err)
		}

		return nil
	}, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get blocks: %w", err)
	}

	return blockInfos, nil
}

func (s *service) WriteBlock(id string, path string, sequence uint64, data []byte) error {
	dataFilePath := filepath.Join(s.opts.Dir, id)
	err := os.WriteFile(dataFilePath, data, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to write data file to path %s: %w", dataFilePath, err)
	}

	blockInfo := BlockInfo{
		ID:           id,
		Sequence:     sequence,
		Length:       uint32(len(data)),
		Path:         path,
		DataFilePath: dataFilePath,
		CRC:          crc32.ChecksumIEEE(data),
	}

	err = s.opts.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Create(&blockInfo).Error
		if err != nil {
			return fmt.Errorf("failed to create block info: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create block info: %w", err)
	}

	_, err = s.opts.NotificationClient.NotifyBlockAdded(context.Background(), &proto.NotifyBlockAddedRequest{
		Host:     s.opts.Host,
		BlockId:  blockInfo.ID,
		Path:     blockInfo.Path,
		Crc:      blockInfo.CRC,
		Sequence: blockInfo.Sequence,
		Length:   blockInfo.Length,
	})
	if err != nil {
		return fmt.Errorf("failed to notify blocks added: %w", err)
	}

	return nil
}

func (s *service) DeleteBlock(id string) error {
	var path string
	err := s.opts.DB.Transaction(func(tx *gorm.DB) error {
		blockInfo := BlockInfo{
			ID: id,
		}
		err := tx.First(&blockInfo).Error
		if err != nil {
			return fmt.Errorf("failed to get block info: %w", err)
		}
		path = blockInfo.DataFilePath
		err = tx.Delete(&blockInfo).Error
		if err != nil {
			return fmt.Errorf("failed to delete block info: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to delete block info: %w", err)
	}

	_, err = s.opts.NotificationClient.NotifyBlockRemoved(context.Background(), &proto.NotifyBlockRemovedRequest{
		Host:    s.opts.Host,
		BlockId: id,
		Path:    path,
	})
	if err != nil {
		return fmt.Errorf("failed to notify blocks removed: %w", err)
	}

	err = os.Remove(path)
	if err != nil {
		return fmt.Errorf("failed to remove data file: %w", err)
	}

	return nil
}

func (s *service) ReadBlock(id string) ([]byte, BlockInfo, error) {
	var data []byte
	var blockInfo BlockInfo
	err := s.opts.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Where("id = ?", id).First(&blockInfo).Error
		if err != nil {
			return fmt.Errorf("failed to get block info: %w", err)
		}

		return nil
	}, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, blockInfo, fmt.Errorf("failed to get block info: %w", err)
	}

	data, err = os.ReadFile(blockInfo.DataFilePath)
	if err != nil {
		return nil, blockInfo, fmt.Errorf("failed to read data file %s: %w", blockInfo.DataFilePath, err)
	}

	if blockInfo.CRC != crc32.ChecksumIEEE(data) {
		return nil, blockInfo, fmt.Errorf("invalid checksum (mismatch)")
	}

	if blockInfo.Length != uint32(len(data)) {
		return nil, blockInfo, fmt.Errorf("invalid length")
	}

	return data, blockInfo, nil
}

func (s *service) Report() error {
	blockInfos, err := s.GetBlocks()
	if err != nil {
		return fmt.Errorf("failed to get blocks: %w", err)
	}
	var allErrors []error

	for _, blockInfo := range blockInfos {
		_, err = s.opts.NotificationClient.NotifyBlockPresent(context.Background(), &proto.NotifyBlockPresentRequest{
			Host:     s.opts.Host,
			BlockId:  blockInfo.ID,
			Path:     blockInfo.Path,
			Crc:      blockInfo.CRC,
			Sequence: blockInfo.Sequence,
			Length:   blockInfo.Length,
		})
		allErrors = append(allErrors, err)
	}

	err = errors.Join(allErrors...)
	if err != nil {
		return fmt.Errorf("failed to report blocks: %w", err)
	}

	return nil
}

func (s *service) HealthCheck() error {
	blockInfos, err := s.GetBlocks()
	if err != nil {
		return fmt.Errorf("failed to get blocks: %w", err)
	}

	var allErrors []error

	for _, blockInfo := range blockInfos {
		path := blockInfo.DataFilePath

		if _, err = os.Stat(path); os.IsNotExist(err) {
			err = s.DeleteBlock(blockInfo.ID)
			if err != nil {
				allErrors = append(allErrors, fmt.Errorf("failed to delete block info: %w", err))
				break
			}

		}
	}

	err = errors.Join(allErrors...)
	if err != nil {
		return fmt.Errorf("failed to health check blocks: %w", err)
	}

	return nil
}

func (s *service) ValidateCRC() error {
	type record struct {
		crc    uint32
		length uint32
	}
	pathRecords := map[string]record{}

	files, err := os.ReadDir(s.opts.Dir)
	if err != nil {
		return fmt.Errorf("cannot read dir %s : %w", s.opts.Dir, err)
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		path := f.Name()
		_, err := f.Info()
		if err != nil {
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			err := os.Remove(path)
			if err != nil {
				continue
			}
		}

		pathRecords[path] = record{
			crc:    crc32.ChecksumIEEE(data),
			length: uint32(len(data)),
		}
	}

	blockInfos, err := s.GetBlocks()
	if err != nil {
		return fmt.Errorf("failed to get blocks: %w", err)
	}

	var allErrors []error

	for _, blockInfo := range blockInfos {
		path := blockInfo.DataFilePath
		blockCRC := blockInfo.CRC
		blockLength := blockInfo.Length

		willDelete := false

		record, found := pathRecords[path]
		if found {
			if record.crc != blockCRC || record.length != blockLength {
				willDelete = true
			}
		} else {
			willDelete = true
		}
		delete(pathRecords, path)

		if willDelete {
			err = s.DeleteBlock(blockInfo.ID)
			allErrors = append(allErrors, err)
		}
	}

	for path := range pathRecords {
		err = os.Remove(path)
		allErrors = append(allErrors, err)
	}

	err = errors.Join(allErrors...)

	if err != nil {
		return fmt.Errorf("failed to validate crc: %w", err)
	}

	return nil
}
