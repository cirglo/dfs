package nameserver

import (
	"fmt"
	"github.com/cirglo.com/dfs/pkg/name"
	"gorm.io/gorm"
)

func CreateSecurityDB(dialector gorm.Dialector) (*gorm.DB, error) {
	db, err := gorm.Open(dialector, &gorm.Config{
		SkipDefaultTransaction:   true,
		DisableNestedTransaction: true,
	})
	if err != nil {
		return nil, fmt.Errorf("could not open database: %w", err)
	}
	if err := db.AutoMigrate(name.User{}, name.Group{}, name.Token{}); err != nil {
		return nil, fmt.Errorf("failed to auto migrate: %w", err)
	}

	return db, nil
}
