package main

import (
	"flag"
	"fmt"
	"github.com/cirglo.com/dfs/pkg/name"
	"github.com/cirglo.com/dfs/pkg/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	logLevelFlag := flag.String("log-level", "info", "Log Level")
	hostFlag := flag.String("host", "localhost", "Node Host")
	portFlag := flag.Int("port", 53035, "Port to listen on")
	dbDriverFlag := flag.String("db-driver", "sqlite", "Database Driver (sqlite, postgres, mysql)")
	dsnFlag := flag.String("dsn", "nameserver.db", "Data Source Name (DSN) for the database")
	dbPoolMaxIdleConnectionsFlag := flag.Int("db-pool-max-idle-connections", 10, "Max Idle Connections in the DB Pool")
	dbPoolMaxOpenConnectionsFlag := flag.Int("db-pool-max-open-connections", 100, "Max Open Connections in the DB Pool")
	dbPoolMaxLifetimeFlag := flag.Duration("db-pool-max-lifetime", 1*time.Hour, "Max Lifetime of Connections in the DB Pool")
	dbPoolMaxIdleTimeFlag := flag.Duration("db-pool-max-idle-time", 10*time.Minute, "Max Lifetime of Connections in the DB Pool")
	tokenExpirationFlag := flag.Duration("token-expiration", 24*time.Hour, "Token Expiration duration")
	numReplicasFlag := flag.Uint("num-replicas", 1, "Number of replicas")
	nodeExpirationFlag := flag.Duration("node-expiration", 15*time.Minute, "Node Expiration duration")
	healingIntervalFlag := flag.Duration("healing-interval", 1*time.Minute, "Healing interval")
	var dialector gorm.Dialector

	flag.Parse()

	log := logrus.New()
	logLevel, err := logrus.ParseLevel(*logLevelFlag)
	if err != nil {
		log.WithError(err).WithField("level", *logLevelFlag).Fatalf("Invalid log level")
	}

	log.SetLevel(logLevel)

	log.WithField("driver", *dbDriverFlag).Info("Opening Database")
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

	log.WithField("driver", *dbDriverFlag).Info("Creating database connection")
	db, err := createDB(dialector)
	if err != nil {
		log.WithError(err).Fatal("Failed to create database")
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.WithError(err).Fatal("Failed to get SQL DB")
	}
	log.Info("Database connection created")

	sqlDB.SetMaxIdleConns(*dbPoolMaxIdleConnectionsFlag)
	sqlDB.SetMaxOpenConns(*dbPoolMaxOpenConnectionsFlag)
	sqlDB.SetConnMaxLifetime(*dbPoolMaxLifetimeFlag)
	sqlDB.SetConnMaxIdleTime(*dbPoolMaxIdleTimeFlag)

	err = sqlDB.Ping()
	if err != nil {
		log.WithError(err).Fatal("Failed to ping database")
	}
	log.Info("Database connection established")

	log.Info("Creating services")
	securityService, err := name.NewSecurityService(name.SecurityServiceOpts{
		Logger:           log,
		DB:               db,
		TokenExperiation: *tokenExpirationFlag,
	})
	if err != nil {
		log.WithError(err).Fatal("Failed to create security service")
	}

	fileService, err := name.NewFileService(name.FileServiceOpts{
		Logger: log,
		DB:     db,
	})
	if err != nil {
		log.WithError(err).Fatal("Failed to create file service")
	}

	log.Info("Creating server")
	server := name.Server{Opts: name.ServerOpts{
		Logger:          log,
		SecurityService: securityService,
		FileService:     fileService}}

	healingService, err := name.NewHealingService(name.HealingOpts{
		Logger:            nil,
		NumReplicas:       *numReplicasFlag,
		FileService:       fileService,
		NodeExpiration:    *nodeExpirationFlag,
		ConnectionFactory: proto.NewInsecureConnectionFactory(),
	})
	if err != nil {
		log.WithError(err).Fatal("Failed to create healing service")
	}

	notificationServer := name.NotificationServer{
		FileService:    fileService,
		HealingService: healingService,
	}

	log.WithField("host", *hostFlag).WithField("port", *portFlag).Info("Starting network listener")
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", *hostFlag, *portFlag))
	if err != nil {
		log.WithError(err).Fatal("Failed to listen")
	}

	grpcServer := grpc.NewServer()
	proto.RegisterNameServer(grpcServer, server)
	proto.RegisterNotificationServer(grpcServer, notificationServer)

	go func() {
		t := time.NewTicker(*healingIntervalFlag)
		for range t.C {
			err := healingService.Heal(time.Now())
			log.WithError(err).Info("Healing failed")
		}
	}()

	log.Info("Starting gRPC server")
	if err := grpcServer.Serve(listener); err != nil {
		log.WithError(err).Fatal("Failed to serve gRPC server")
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
		name.BlockInfo{})
	if err != nil {
		return nil, fmt.Errorf("failed to auto migrate: %w", err)
	}

	return db, nil
}
