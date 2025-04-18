package name_test

import (
	"github.com/cirglo.com/dfs/pkg/name"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
	"time"
)

func createSecurityDB(t *testing.T) *gorm.DB {
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
		name.Permission{})
	assert.NoError(t, err)

	return db
}

func TestNewSecurityService(t *testing.T) {
	logger := logrus.New()
	db := createSecurityDB(t)
	opts := name.SecurityServiceOpts{
		Logger:           logger,
		DB:               db,
		TokenExperiation: 1 * time.Hour,
	}
	service, err := name.NewSecurityService(opts)
	assert.NoError(t, err)
	assert.NotNil(t, service)
}
