package nameserver

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/cirglo.com/dfs/pkg/name"
	"github.com/cirglo.com/dfs/pkg/proto"
	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"strings"
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
	etcdIntervalFlag := flag.Duration("etcd-interval", 1*time.Minute, "ETCD Interval")
	etcdTimeoutFlag := flag.Duration("etcd-timeout", 5*time.Second, "ETCD Timeout")
	etcdEndpointsFlag := flag.String("etcd-endpoints", "localhost:2379", "ETCD Endpoints")
	etcdUsernameFlag := flag.String("etcd-username", "", "ETCD Username")
	etcPasswordFlag := flag.String("etcd-password", "", "ETCD Password")

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

	client, err := clientv3.New(clientv3.Config{
		Endpoints: strings.Split(*etcdEndpointsFlag, ","),
		Username:  *etcdUsernameFlag,
		Password:  *etcPasswordFlag,
	})

	etcdOpts := name.EtcdOpts{
		ContextFactory: func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), *etcdTimeoutFlag)
		},
		Client: client,
	}

	etcd, err := name.NewEtcd(etcdOpts)
	if err != nil {
		log.WithError(err).Fatal("Failed to create etcd client")
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

	err = updateNodeInfo(etcd, fileService)
	if err != nil {
		log.WithError(err).Fatal("Failed to gather node info from etcd")
	}

	go func() {
		ticker := time.NewTicker(*etcdIntervalFlag)
		for range ticker.C {
			err := updateNodeInfo(etcd, fileService)
			if err != nil {
				log.WithError(err).Fatal("Failed to update node info")
			}
		}
	}()
}

func updateNodeInfo(etcd name.Etcd, fileService name.FileService) error {
	nodeInfos, err := etcd.Gather()
	if err != nil {
		return fmt.Errorf("Failed to gather node info from etcd")
	}

	var allErrors []error
	var paths map[string]map[uint64]*name.BlockInfo

	for _, nodeInfo := range nodeInfos {
		host := nodeInfo.Host
		port := nodeInfo.Port
		location := nodeInfo.Location

		conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", host, port))
		if err != nil {
			allErrors = append(allErrors, fmt.Errorf("failed to create gRPC client: %w", err))
			continue
		}
		nodeClient := proto.NewNodeClient(conn)
		response, err := nodeClient.GetBlockInfos(context.Background(), &proto.GetBlockInfosRequest{})
		if err != nil {
			allErrors = append(allErrors, fmt.Errorf("failed to get block info: %w", err))
			continue
		}

		for _, protoBlockInfo := range response.BlockInfos {
			if _, ok := paths[protoBlockInfo.Path]; !ok {
				paths[protoBlockInfo.Path] = map[uint64]*name.BlockInfo{}
			}
			if _, ok := paths[protoBlockInfo.Path][protoBlockInfo.BlockId]; ok {
				paths[protoBlockInfo.Path][protoBlockInfo.BlockId] = &name.BlockInfo{
					ID:        protoBlockInfo.BlockId,
					Length:    protoBlockInfo.Length,
					Sequence:  protoBlockInfo.Sequence,
					Locations: []*name.Location{},
				}
			}
			blockInfo := paths[protoBlockInfo.Path][protoBlockInfo.BlockId]
			blockInfo.Locations = append(blockInfo.Locations, &name.Location{
				Hostname: host,
				Port:     port,
				Value:    location,
			})
		}

		for path, blockIdMap := range paths {
			blockInfos := []name.BlockInfo{}
			for _, blockInfo := range blockIdMap {
				blockInfos = append(blockInfos, *blockInfo)
			}
			err := fileService.UpdateBlockInfos(name.NewRootPrincipal(), path, blockInfos)
			if err != nil {
				allErrors = append(allErrors, fmt.Errorf("failed to update block infos: %w", err))
			}
		}
	}

	if len(allErrors) > 0 {
		return errors.Join(allErrors...)
	}

	return nil
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
