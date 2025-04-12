package node

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/cirglo.com/dfs/pkg/proto"
	"google.golang.org/grpc"
	"gorm.io/gorm"
	"hash/crc32"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
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
	ID           string `gorm:"primaryKey; not null"`
	Sequence     uint64 `gorm:"not null;uniqueIndex:idx_block_info'"`
	Length       uint32 `gorm:"not null;"`
	Path         string `gorm:"not null;not empty;"`
	DataFilePath string `gorm:"not null;uniqueIndex:idx_block_info;"`
	CRC          uint32 `gorm:"not null;"`
}
type ServiceOpts struct {
	Logger                  *logrus.Logger
	NameNodeHost            string
	Host                    string
	DB                      *gorm.DB
	Dir                     fs.FileInfo
	ClientConnectionFactory func(destination string) (*grpc.ClientConn, error)
	ReportInterval          time.Duration
	HealthCheckInterval     time.Duration
	ValidateCRCInterval     time.Duration
}

func (o *ServiceOpts) Validate() error {
	if o.Logger == nil {
		return fmt.Errorf("logger is required")
	}

	if o.DB == nil {
		return fmt.Errorf("db is required")
	}

	if o.Dir == nil {
		return fmt.Errorf("dir is required")
	}

	if !o.Dir.IsDir() {
		return fmt.Errorf("dir is not a directory: %s", o.Dir)
	}

	if o.NameNodeHost == "" {
		return fmt.Errorf("name node host is required")
	}

	if o.Host == "" {
		return fmt.Errorf("host is required")
	}

	if o.ReportInterval == 0 {
		return fmt.Errorf("report interval is required")
	}

	if o.HealthCheckInterval == 0 {
		return fmt.Errorf("health check interval is required")
	}

	if o.ValidateCRCInterval == 0 {
		return fmt.Errorf("validate CRC interval is required")
	}

	if o.ClientConnectionFactory == nil {
		return fmt.Errorf("client connection factory is required")
	}

	return nil
}

type service struct {
	opts ServiceOpts
}

func NewService(opts ServiceOpts) (Service, error) {
	err := opts.Validate()
	if err != nil {
		return nil, fmt.Errorf("options are not valid: %w", err)
	}

	s := service{opts: opts}

	err = s.report()
	if err != nil {
		return nil, fmt.Errorf("report failed: %w", err)
	}

	err = s.healthCheck()
	if err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}

	err = s.validateCRC()
	if err != nil {
		return nil, fmt.Errorf("validate CRC failed: %w", err)
	}

	go func() {
		ticker := time.NewTicker(opts.ReportInterval)
		for range ticker.C {
			err := s.report()
			if err != nil {
				s.opts.Logger.WithError(err).Fatal("report failed")
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(opts.HealthCheckInterval)
		for range ticker.C {
			err := s.healthCheck()
			if err != nil {
				s.opts.Logger.WithError(err).Fatal("health check failed")
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(opts.ValidateCRCInterval)
		for range ticker.C {
			err := s.validateCRC()
			if err != nil {
				s.opts.Logger.WithError(err).Fatal("validate CRC failed")
			}
		}
	}()

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

func (s *service) WriteBlock(blockInfo BlockInfo, data []byte) error {
	if blockInfo.CRC != crc32.ChecksumIEEE(data) {
		return fmt.Errorf("invalid checksum (mismatch)")
	}
	if blockInfo.Length != uint32(len(data)) {
		return fmt.Errorf("invalid length")
	}

	path := filepath.Join(s.opts.Dir.Name(), fmt.Sprintf("%s.data", uuid.New().String()))
	err := os.WriteFile(path, data, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to write data file: %w", err)
	}

	err = s.opts.DB.Transaction(func(tx *gorm.DB) error {
		blockInfo.DataFilePath = path
		err := tx.Create(&blockInfo).Error
		if err != nil {
			return fmt.Errorf("failed to create block info: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create block info: %w", err)
	}

	conn, err := s.opts.ClientConnectionFactory(s.opts.NameNodeHost)
	if err != nil {
		return fmt.Errorf("failed to create client connection: %w", err)
	}
	defer conn.Close()
	client := proto.NewNameInternalClient(conn)

	_, err = client.NotifyBlocksAdded(context.Background(), &proto.BlockInfoReport{
		Host: s.opts.Host,
		BlockInfos: []*proto.BlockInfoItem{{
			BlockId:  blockInfo.ID,
			Path:     blockInfo.Path,
			Crc:      blockInfo.CRC,
			Sequence: blockInfo.Sequence,
			Length:   blockInfo.Length,
		}},
	})
	if err != nil {
		return fmt.Errorf("failed to notify blocks added: %w", err)
	}

	return nil
}

func (s *service) DeleteBlock(id string) error {
	var path string
	err := s.opts.DB.Transaction(func(tx *gorm.DB) error {
		blockInfo := BlockInfo{}
		err := tx.Model(&blockInfo).Where("id = ?", id).Error
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

	conn, err := s.opts.ClientConnectionFactory(s.opts.NameNodeHost)
	if err != nil {
		return fmt.Errorf("failed to create client connection: %w", err)
	}
	defer conn.Close()
	client := proto.NewNameInternalClient(conn)
	_, err = client.NotifyBlocksRemoved(context.Background(), &proto.BlockInfoReport{
		Host: s.opts.Host,
		BlockInfos: []*proto.BlockInfoItem{{
			BlockId: id,
			Path:    path,
		}},
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
		err := tx.Model(&blockInfo).Where("id = ?", id).Error
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
		return nil, blockInfo, fmt.Errorf("failed to read data file: %w", err)
	}

	if blockInfo.CRC != crc32.ChecksumIEEE(data) {
		return nil, blockInfo, fmt.Errorf("invalid checksum (mismatch)")
	}

	if blockInfo.Length != uint32(len(data)) {
		return nil, blockInfo, fmt.Errorf("invalid length")
	}

	return data, blockInfo, nil
}

func (s *service) report() error {
	protoBlockInfos := []*proto.BlockInfoItem{}

	err := s.opts.DB.Transaction(func(tx *gorm.DB) error {
		blockInfos := []BlockInfo{}
		err := tx.Find(&blockInfos).Error
		if err != nil {
			return fmt.Errorf("failed to get block ids: %w", err)
		}

		for _, blockInfo := range blockInfos {
			protoBlockInfos = append(protoBlockInfos, &proto.BlockInfoItem{
				BlockId:  blockInfo.ID,
				Path:     blockInfo.Path,
				Crc:      blockInfo.CRC,
				Sequence: blockInfo.Sequence,
				Length:   blockInfo.Length,
			})
		}

		return nil
	}, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to health check block info: %w", err)
	}

	conn, err := s.opts.ClientConnectionFactory(s.opts.NameNodeHost)
	if err != nil {
		return fmt.Errorf("failed to create client connection: %w", err)
	}
	defer conn.Close()
	client := proto.NewNameInternalClient(conn)

	_, err = client.NotifyBlocksRemoved(context.Background(), &proto.BlockInfoReport{
		Host:       s.opts.Host,
		BlockInfos: protoBlockInfos,
	})
	if err != nil {
		return fmt.Errorf("failed to report: %w", err)
	}

	return nil
}

func (s *service) healthCheck() error {
	deletedBlockInfoReport := &proto.BlockInfoReport{
		Host:       s.opts.Host,
		BlockInfos: []*proto.BlockInfoItem{},
	}
	pathsToDelete := map[string]bool{}

	err := s.opts.DB.Transaction(func(tx *gorm.DB) error {
		blockInfos := []BlockInfo{}
		err := tx.Find(&blockInfos).Error
		if err != nil {
			return fmt.Errorf("failed to get block ids: %w", err)
		}

		logEntry := s.opts.Logger.WithField("block-infos-count", len(blockInfos))

		for _, blockInfo := range blockInfos {
			logEntry = logEntry.WithField("block-info", blockInfo)
			path := blockInfo.DataFilePath

			if _, err := os.Stat(path); os.IsNotExist(err) {
				err := tx.Delete(&blockInfo).Error
				if err != nil {
					return fmt.Errorf("failed to delete block info: %w", err)
				}
				deletedBlockInfoReport.BlockInfos = append(
					deletedBlockInfoReport.BlockInfos,
					&proto.BlockInfoItem{
						BlockId: blockInfo.ID,
					})
				pathsToDelete[path] = true
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to health check block info: %w", err)
	}

	if len(deletedBlockInfoReport.GetBlockInfos()) > 0 {
		conn, err := s.opts.ClientConnectionFactory(s.opts.NameNodeHost)
		if err != nil {
			return fmt.Errorf("failed to create client connection: %w", err)
		}
		defer conn.Close()
		client := proto.NewNameInternalClient(conn)

		_, err = client.NotifyBlocksRemoved(context.Background(), deletedBlockInfoReport)
	}

	if len(pathsToDelete) > 0 {
		s.opts.Logger.Warn("deleting bad blocks")
		for path := range pathsToDelete {
			err := os.Remove(path)
			if err != nil {
				s.opts.Logger.WithError(err).Error("failed to remove file")
			}
		}
		s.opts.Logger.Warn("finished deleting bad blocks")
	}

	return nil
}

func (s *service) validateCRC() error {
	pathCrcs := map[string]uint32{}

	files, err := os.ReadDir(s.opts.Dir.Name())
	if err != nil {
		return fmt.Errorf("cannot read dir %s : %w", s.opts.Dir, err)
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		path := f.Name()

		if !strings.HasSuffix(path, ".data") {
			continue
		}

		logEntry := s.opts.Logger.WithField("file-path", path)

		_, err := f.Info()
		if err != nil {
			logEntry.WithError(err).Error("cannot read file info")
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			logEntry.WithError(err).Error("cannot read file")
			err := os.Remove(path)
			if err != nil {
				logEntry.WithError(err).Error("failed to remove file")
			}
			continue
		}

		pathCrcs[path] = crc32.ChecksumIEEE(data)
	}

	pathsToDelete := map[string]bool{}
	deletedBlockInfoReport := &proto.BlockInfoReport{
		Host:       s.opts.Host,
		BlockInfos: []*proto.BlockInfoItem{},
	}

	err = s.opts.DB.Transaction(func(tx *gorm.DB) error {
		blockInfos := []BlockInfo{}
		err := tx.Find(&blockInfos).Error
		if err != nil {
			return fmt.Errorf("failed to get block ids: %w", err)
		}

		logEntry := s.opts.Logger.WithField("block-infos-count", len(blockInfos))

		for _, blockInfo := range blockInfos {
			logEntry = logEntry.WithField("block-info", blockInfos)
			path := blockInfo.DataFilePath

			if _, ok := pathCrcs[path]; !ok {
				err := tx.Delete(&blockInfo).Error
				if err != nil {
					return fmt.Errorf("failed to delete block info: %w", err)
				}

				deletedBlockInfoReport.BlockInfos = append(
					deletedBlockInfoReport.BlockInfos,
					&proto.BlockInfoItem{
						BlockId: blockInfo.ID,
					})
			}

			if pathCrcs[path] != blockInfo.CRC {
				err := tx.Delete(&blockInfo).Error
				if err != nil {
					return fmt.Errorf("failed to delete block info: %w", err)
				}
				deletedBlockInfoReport.BlockInfos = append(
					deletedBlockInfoReport.BlockInfos,
					&proto.BlockInfoItem{
						BlockId: blockInfo.ID,
					})
				pathsToDelete[path] = true
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to delete block info: %w", err)
	}

	for path := range pathsToDelete {
		err := os.Remove(path)
		if err != nil {
			s.opts.Logger.WithError(err).WithField("file", path).Error("failed to remove file")
		}
	}

	if len(deletedBlockInfoReport.GetBlockInfos()) > 0 {
		conn, err := s.opts.ClientConnectionFactory(s.opts.NameNodeHost)
		if err != nil {
			return fmt.Errorf("failed to create client connection: %w", err)
		}
		defer conn.Close()
		client := proto.NewNameInternalClient(conn)

		_, err = client.NotifyBlocksRemoved(context.Background(), deletedBlockInfoReport)
	}

	return nil
}
