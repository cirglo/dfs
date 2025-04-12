package nameserver

import (
	"flag"
	"fmt"
	"github.com/cirglo.com/dfs/pkg/name"
	"github.com/sirupsen/logrus"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	logLevelFlag := flag.String("log-level", "info", "Log Level")
	dbDriverFlag := flag.String("db-driver", "sqlite", "Database Driver (sqlite, postgres, mysql)")
	dsnFlag := flag.String("dsn", "nameserver.db", "Data Source Name (DSN) for the database")
	dbPoolMaxIdleConnectionsFlag := flag.Int("db-pool-max-idle-connections", 10, "Max Idle Connections in the DB Pool")
	dbPoolMaxOpenConnectionsFlag := flag.Int("db-pool-max-open-connections", 100, "Max Open Connections in the DB Pool")
	dbPoolMaxLifetimeFlag := flag.Duration("db-pool-max-lifetime", 1*time.Hour, "Max Lifetime of Connections in the DB Pool")
	dbPoolMaxIdleTimeFlag := flag.Duration("db-pool-max-idle-time", 10*time.Minute, "Max Lifetime of Connections in the DB Pool")
	tokenExpirationFlag := flag.Duration("token-expiration", 24*time.Hour, "Token Expiration duration")
	flag.Int("port", 50051, "Name Port")

	var dialector gorm.Dialector

	flag.Parse()

	log := logrus.New()
	logLevel, err := logrus.ParseLevel(*logLevelFlag)
	if err != nil {
		log.WithError(err).WithField("level", *logLevelFlag).Fatalf("Invalid log level")
	}

	log.SetLevel(logLevel)

	switch *dbDriverFlag {
	case "sqlite":
		dialector = sqlite.Open(*dsnFlag)
	case "postgres":
		dialector = postgres.Open(*dsnFlag)
	case "mysql":
		dialector = mysql.Open(*dsnFlag)
	default:
		log.Fatalf("Invalid database driver: %s", *dbDriverFlag)
	}

	db, err := createDB(dialector)
	if err != nil {
		log.WithError(err).Fatal("Failed to create database")
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.WithError(err).Fatal("Failed to get SQL DB")
	}

	sqlDB.SetMaxIdleConns(*dbPoolMaxIdleConnectionsFlag)
	sqlDB.SetMaxOpenConns(*dbPoolMaxOpenConnectionsFlag)
	sqlDB.SetConnMaxLifetime(*dbPoolMaxLifetimeFlag)
	sqlDB.SetConnMaxIdleTime(*dbPoolMaxIdleTimeFlag)

	err = sqlDB.Ping()
	if err != nil {
		log.WithError(err).Fatal("Failed to ping database")
	}

	securityService, err := name.NewSecurityService(name.SecurityServiceOpts{
		Logger:           log,
		DB:               db,
		TokenExperiation: *tokenExpirationFlag,
	})
	if err != nil {
		log.WithError(err).Fatal("Failed to create security service")
	}

	_, err = securityService.GetAllUsers()
	if err != nil {
		log.WithError(err).Fatal("Failed to get all users")
	}

	fileService, err := name.NewFileService(name.FileServiceOpts{
		Logger: log,
		DB:     db,
	})
	if err != nil {
		log.WithError(err).Fatal("Failed to create file service")
	}
}

func createDB(dialector gorm.Dialector) (*gorm.DB, error) {
	db, err := gorm.Open(dialector, &gorm.Config{
		SkipDefaultTransaction:   true,
		DisableNestedTransaction: true,
	})
	if err != nil {
		return nil, fmt.Errorf("could not open database: %w", err)
	}
	err = db.AutoMigrate(
		name.User{},
		name.Group{},
		name.Token{},
		name.Permissions{},
		name.FileInfo{},
		name.Permission{},
		name.DirInfo{},
		name.Location{},
		name.BlockInfo{})
	if err != nil {
		return nil, fmt.Errorf("failed to auto migrate: %w", err)
	}

	return db, nil
}
