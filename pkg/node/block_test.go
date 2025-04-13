package node_test

import (
	"github.com/cirglo.com/dfs/pkg/mocks"
	"github.com/cirglo.com/dfs/pkg/node"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"testing"
)

func createDB(t *testing.T) *gorm.DB {
	dialector := sqlite.Open(":memory:")
	db, err := gorm.Open(dialector, &gorm.Config{
		SkipDefaultTransaction:   true,
		DisableNestedTransaction: true,
	})
	assert.NoError(t, err)
	err = db.AutoMigrate(node.BlockInfo{})
	assert.NoError(t, err)

	return db
}

func createLogger(t *testing.T) *logrus.Logger {
	log := logrus.New()
	log.SetLevel(logrus.FatalLevel)

	return log
}

func createDir(t *testing.T) string {
	path := t.TempDir()
	dirInfo, err := os.Stat(path)
	assert.NoError(t, err)
	assert.True(t, dirInfo.IsDir())

	absPath, err := filepath.Abs(path)
	assert.NoError(t, err)

	return absPath
}

func TestBlockService_Write_Read_Delete_Block(t *testing.T) {
	log := createLogger(t)
	db := createDB(t)
	dir := createDir(t)
	nameClient := mocks.NewNameInternalClient(t)
	opts := node.BlockServiceOpts{
		Logger:     log,
		Host:       "whoof:2345",
		DB:         db,
		Dir:        dir,
		NameClient: nameClient,
	}
	service, err := node.NewBlockService(opts)
	assert.NoError(t, err)

	blocks, err := service.GetBlocks()
	assert.NoError(t, err)
	assert.Len(t, blocks, 0)

	id := uuid.New().String()
	sequence := uint64(0)
	path := "/hello.txt"
	data := []byte("hello")

	nameClient.EXPECT().NotifyBlocksAdded(mock.Anything, mock.Anything).Return(nil, nil)

	err = service.WriteBlock(id, path, sequence, data)
	assert.NoError(t, err)

	blocks, err = service.GetBlocks()
	assert.NoError(t, err)
	assert.Len(t, blocks, 1)
	assert.Equal(t, id, blocks[0].ID)
	assert.NotEqual(t, uint32(0), blocks[0].CRC)
	assert.Equal(t, uint64(0), blocks[0].Sequence)
	assert.Equal(t, uint32(len(data)), blocks[0].Length)
	assert.NotEmpty(t, blocks[0].DataFilePath)

	d, bi, err := service.ReadBlock(id)
	assert.NoError(t, err)
	assert.Equal(t, data, d)
	assert.Equal(t, id, bi.ID)
	assert.NotEqual(t, uint32(0), bi.CRC)
	assert.Equal(t, uint64(0), bi.Sequence)
	assert.Equal(t, uint32(len(data)), bi.Length)
	assert.NotEmpty(t, bi.DataFilePath)

	nameClient.EXPECT().NotifyBlocksRemoved(mock.Anything, mock.Anything).Return(nil, nil)

	err = service.DeleteBlock(id)
	assert.NoError(t, err)

	blocks, err = service.GetBlocks()
	assert.NoError(t, err)
	assert.Len(t, blocks, 0)

	nameClient.AssertExpectations(t)
}
