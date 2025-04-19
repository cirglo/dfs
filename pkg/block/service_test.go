package block_test

import (
	"fmt"
	"github.com/cirglo.com/dfs/pkg/block"
	"github.com/cirglo.com/dfs/pkg/mocks"
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
	err = db.AutoMigrate(block.BlockInfo{})
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
	notificationClient := mocks.NewNotificationClient(t)
	opts := block.Opts{
		Logger:             log,
		Host:               "whoof:2345",
		DB:                 db,
		Dir:                dir,
		NotificationClient: notificationClient,
	}
	service, err := block.NewService(opts)
	assert.NoError(t, err)

	blocks, err := service.GetBlocks()
	assert.NoError(t, err)
	assert.Len(t, blocks, 0)

	id := uuid.New().String()
	sequence := uint64(0)
	path := "/hello.txt"
	data := []byte("hello")

	notificationClient.EXPECT().
		NotifyBlockAdded(mock.Anything, mock.Anything).
		Return(nil, nil).
		Once()

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

	notificationClient.EXPECT().
		NotifyBlockRemoved(mock.Anything, mock.Anything).
		Return(nil, nil).
		Once()

	err = service.DeleteBlock(id)
	assert.NoError(t, err)

	blocks, err = service.GetBlocks()
	assert.NoError(t, err)
	assert.Len(t, blocks, 0)

	notificationClient.AssertExpectations(t)
}

func TestBlockService_GetBlockIds(t *testing.T) {
	log := createLogger(t)
	db := createDB(t)
	dir := createDir(t)
	notificationClient := mocks.NewNotificationClient(t)
	opts := block.Opts{
		Logger:             log,
		Host:               "whoof:2345",
		DB:                 db,
		Dir:                dir,
		NotificationClient: notificationClient,
	}
	service, err := block.NewService(opts)
	assert.NoError(t, err)

	notificationClient.EXPECT().NotifyBlockAdded(mock.Anything, mock.Anything).Return(nil, nil).Once()
	// Write a block
	id := uuid.New().String()
	err = service.WriteBlock(id, "/test.txt", 1, []byte("test data"))
	assert.NoError(t, err)

	// Get block IDs
	blockIds, err := service.GetBlockIds()
	assert.NoError(t, err)
	assert.Len(t, blockIds, 1)
	assert.Equal(t, id, blockIds[0])

	notificationClient.AssertExpectations(t)
}

func TestBlockService_HealthCheck(t *testing.T) {
	log := createLogger(t)
	db := createDB(t)
	dir := createDir(t)
	notificationClient := mocks.NewNotificationClient(t)
	opts := block.Opts{
		Logger:             log,
		Host:               "whoof:2345",
		DB:                 db,
		Dir:                dir,
		NotificationClient: notificationClient,
	}
	service, err := block.NewService(opts)
	assert.NoError(t, err)

	notificationClient.EXPECT().NotifyBlockAdded(mock.Anything, mock.Anything).Return(nil, nil).Once()
	// Write a block
	id := uuid.New().String()
	err = service.WriteBlock(id, "/test.txt", 1, []byte("test data"))
	assert.NoError(t, err)

	// Simulate missing file
	block, err := service.GetBlocks()
	assert.NoError(t, err)
	assert.Len(t, block, 1)
	err = os.Remove(block[0].DataFilePath)
	assert.NoError(t, err)

	// Run health check
	notificationClient.EXPECT().
		NotifyBlockRemoved(mock.Anything, mock.Anything).
		Return(nil, nil).
		Once()
	err = service.HealthCheck()
	assert.NoError(t, err)

	// Verify block is removed
	blocks, err := service.GetBlocks()
	assert.NoError(t, err)
	assert.Len(t, blocks, 0)

	notificationClient.AssertExpectations(t)
}

func TestBlockService_ValidateCRC(t *testing.T) {
	log := createLogger(t)
	db := createDB(t)
	dir := createDir(t)
	notificationClient := mocks.NewNotificationClient(t)
	opts := block.Opts{
		Logger:             log,
		Host:               "whoof:2345",
		DB:                 db,
		Dir:                dir,
		NotificationClient: notificationClient,
	}
	service, err := block.NewService(opts)
	assert.NoError(t, err)

	notificationClient.EXPECT().NotifyBlockAdded(mock.Anything, mock.Anything).Return(nil, nil).Once()
	// Write a block
	id := uuid.New().String()
	err = service.WriteBlock(id, "/test.txt", 1, []byte("test data"))
	assert.NoError(t, err)

	// Corrupt the file
	block, err := service.GetBlocks()
	assert.NoError(t, err)
	assert.Len(t, block, 1)
	err = os.WriteFile(block[0].DataFilePath, []byte("corrupted data"), os.ModePerm)
	assert.NoError(t, err)

	// Run CRC validation
	notificationClient.EXPECT().
		NotifyBlockRemoved(mock.Anything, mock.Anything).
		Return(nil, nil).
		Once()
	err = service.ValidateCRC()
	assert.NoError(t, err)

	// Verify block is removed
	blocks, err := service.GetBlocks()
	assert.NoError(t, err)
	assert.Len(t, blocks, 0)

	notificationClient.AssertExpectations(t)
}

func TestBlockService_WriteBlock_EmptyID(t *testing.T) {
	log := createLogger(t)
	db := createDB(t)
	dir := createDir(t)
	notificationClient := mocks.NewNotificationClient(t)
	opts := block.Opts{
		Logger:             log,
		Host:               "whoof:2345",
		DB:                 db,
		Dir:                dir,
		NotificationClient: notificationClient,
	}
	service, err := block.NewService(opts)
	assert.NoError(t, err)

	err = service.WriteBlock("", "/test.txt", 1, []byte("test data"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "block id is empty")

	notificationClient.AssertExpectations(t)
}

func TestBlockService_WriteBlock_DuplicateID(t *testing.T) {
	log := createLogger(t)
	db := createDB(t)
	dir := createDir(t)
	notificationClient := mocks.NewNotificationClient(t)
	opts := block.Opts{
		Logger:             log,
		Host:               "whoof:2345",
		DB:                 db,
		Dir:                dir,
		NotificationClient: notificationClient,
	}
	service, err := block.NewService(opts)
	assert.NoError(t, err)

	notificationClient.EXPECT().NotifyBlockAdded(mock.Anything, mock.Anything).Return(nil, nil).Once()
	id := uuid.New().String()
	err = service.WriteBlock(id, "/test.txt", 1, []byte("test data"))
	assert.NoError(t, err)

	err = service.WriteBlock(id, "/test2.txt", 2, []byte("test data 2"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create block info")

	notificationClient.AssertExpectations(t)
}

func TestBlockService_WriteBlock_InvalidPath(t *testing.T) {
	log := createLogger(t)
	db := createDB(t)
	dir := createDir(t)
	notificationClient := mocks.NewNotificationClient(t)
	opts := block.Opts{
		Logger:             log,
		Host:               "whoof:2345",
		DB:                 db,
		Dir:                dir,
		NotificationClient: notificationClient,
	}
	service, err := block.NewService(opts)
	assert.NoError(t, err)

	err = service.WriteBlock(uuid.New().String(), "", 1, []byte("test data"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path is empty")
}

func TestBlockService_NotificationError(t *testing.T) {
	log := createLogger(t)
	db := createDB(t)
	dir := createDir(t)
	notificationClient := mocks.NewNotificationClient(t)
	opts := block.Opts{
		Logger:             log,
		Host:               "whoof:2345",
		DB:                 db,
		Dir:                dir,
		NotificationClient: notificationClient,
	}
	service, err := block.NewService(opts)
	assert.NoError(t, err)

	id := uuid.New().String()
	notificationClient.EXPECT().
		NotifyBlockAdded(mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("notification error")).
		Once()

	err = service.WriteBlock(id, "/test.txt", 1, []byte("test data"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to notify blocks added")
}

func TestBlockService_MissingDirectory(t *testing.T) {
	log := createLogger(t)
	db := createDB(t)
	dir := createDir(t)
	os.RemoveAll(dir) // Remove the directory to simulate missing directory

	notificationClient := mocks.NewNotificationClient(t)
	opts := block.Opts{
		Logger:             log,
		Host:               "whoof:2345",
		DB:                 db,
		Dir:                dir,
		NotificationClient: notificationClient,
	}
	_, err := block.NewService(opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not stat dir")
}
