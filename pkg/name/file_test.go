package name_test

import (
	"github.com/cirglo.com/dfs/pkg/name"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
)

func createDB(t *testing.T) *gorm.DB {
	dialector := sqlite.Open(":memory:")
	db, err := gorm.Open(dialector, &gorm.Config{
		SkipDefaultTransaction:   true,
		DisableNestedTransaction: true,
	})
	assert.NoError(t, err)
	err = db.AutoMigrate(
		name.User{},
		name.Group{},
		name.Token{},
		name.Permissions{},
		name.FileInfo{},
		name.Permission{},
		name.BlockInfo{},
		name.Location{})
	assert.NoError(t, err)

	return db
}

func createLogger(t *testing.T) *logrus.Logger {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	return log
}

func TestFileService_GetRootDir(t *testing.T) {
	log := createLogger(t)
	db := createDB(t)
	p := name.NewRootPrincipal()

	service, err := name.NewFileService(name.FileServiceOpts{
		Logger: log,
		DB:     db,
	})
	assert.NoError(t, err)

	dir, err := service.Stat(p, "/")
	assert.NoError(t, err)
	assert.Equal(t, "", dir.Name)
	assert.Nil(t, dir.ParentID)
	assert.Len(t, dir.Children, 0)
}

func TestFileService_CreateFile(t *testing.T) {
	log := createLogger(t)
	db := createDB(t)
	p := name.NewRootPrincipal()
	perms := name.Permissions{
		Owner: "joe",
		Group: "staff",
		OwnerPermission: name.Permission{
			Read:   true,
			Write:  true,
			Delete: true,
		},
		GroupPermission: name.Permission{
			Read:   true,
			Write:  false,
			Delete: true,
		},
		OtherPermission: name.Permission{
			Read:   false,
			Write:  false,
			Delete: true,
		},
	}

	service, err := name.NewFileService(name.FileServiceOpts{
		Logger: log,
		DB:     db,
	})
	assert.NoError(t, err)

	dirs, err := service.List(p, "/")
	assert.NoError(t, err)
	assert.Len(t, dirs, 0)

	rootFi, err := service.Stat(p, "/")
	assert.NoError(t, err)
	assert.Equal(t, "", rootFi.Name)
	assert.True(t, rootFi.IsDir)
	assert.Nil(t, rootFi.ParentID)

	fi, err := service.CreateFile(p, "/hello.txt", perms)
	assert.NoError(t, err)
	assert.Equal(t, "hello.txt", fi.Name)
	assert.Equal(t, perms, fi.Permissions)

	dirs, err = service.List(p, "/")
	assert.NoError(t, err)
	assert.Len(t, dirs, 1)

	fi, err = service.Stat(p, "/hello.txt")
	assert.NoError(t, err)
	assert.Equal(t, "hello.txt", fi.Name)
	assert.False(t, fi.IsDir)
	assert.Equal(t, rootFi.ID, *fi.ParentID)

	err = service.DeleteFile(p, "/hello.txt")
	assert.NoError(t, err)

	dirs, err = service.List(p, "/")
	assert.NoError(t, err)
	assert.Len(t, dirs, 0)
}

func TestFileService_GetAllBlockInfos_EmptyDB(t *testing.T) {
	log := createLogger(t)
	db := createDB(t)

	service, err := name.NewFileService(name.FileServiceOpts{
		Logger: log,
		DB:     db,
	})
	assert.NoError(t, err)

	blockInfos, err := service.GetAllBlockInfos()
	assert.NoError(t, err)
	assert.Len(t, blockInfos, 0)
}

func TestFileService_NodeRemoved(t *testing.T) {
	log := createLogger(t)
	db := createDB(t)

	service, err := name.NewFileService(name.FileServiceOpts{
		Logger: log,
		DB:     db,
	})
	assert.NoError(t, err)

	// Test removing a node that doesn't exist
	err = service.NodeRemoved("nonexistent-node")
	assert.NoError(t, err)

	// Add a node and then remove it
	err = db.Create(&name.BlockInfo{
		ID: "block1",
		Locations: []name.Location{
			{Host: "host1"},
		},
	}).Error
	assert.NoError(t, err)

	err = service.NodeRemoved("host1")
	assert.NoError(t, err)

	// Verify the node was removed
	var blockInfo name.BlockInfo
	err = db.First(&blockInfo, "id = ?", "block1").Error
	assert.NoError(t, err)
	assert.Len(t, blockInfo.Locations, 0)
}
