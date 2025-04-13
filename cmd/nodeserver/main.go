package main

import (
	"flag"
	"fmt"
	"github.com/cirglo.com/dfs/pkg/node"
	"github.com/cirglo.com/dfs/pkg/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net"
	"os"
	"time"
)

func main() {
	logLevelFlag := flag.String("log-level", "info", "Log Level")
	nameNodeFlag := flag.String("name-node", "localhost:2379", "Name Node Address")
	dirFlag := flag.String("dir", "./", "Node Directory")
	dsnFlag := flag.String("dsn", "nodeserver.db", "Data Source Name (DSN) for the database")
	hostFlag := flag.String("host", "localhost:50051", "Node Host")
	reportIntervalFlag := flag.Duration("report-interval", 10*time.Minute, "Report Interval")
	healthCheckIntervalFlag := flag.Duration("health-check-interval", 1*time.Hour, "Health Check Interval")
	crcCheckIntervalFlag := flag.Duration("crc-check-interval", 24*time.Hour, "CRC Check Interval")

	flag.Parse()

	log := logrus.New()
	logLevel, err := logrus.ParseLevel(*logLevelFlag)
	if err != nil {
		log.WithError(err).WithField("level", *logLevelFlag).Fatalf("Invalid log level")
	}

	log.SetLevel(logLevel)

	dir, err := os.Stat(*dirFlag)
	if err != nil {
		log.WithError(err).WithField("dir", *dirFlag).Fatal("Directory does not exist")
	}

	db, err := createDB(sqlite.Open(*dsnFlag))
	if err != nil {
		log.WithError(err).Fatal("Failed to create database")
	}

	serviceOpts := node.ServiceOpts{
		Logger:       log,
		Host:         *hostFlag,
		NameNodeHost: *nameNodeFlag,
		DB:           db,
		Dir:          dir,
		ClientConnectionFactory: func(destination string) (*grpc.ClientConn, error) {
			return grpc.NewClient(
				destination,
				grpc.WithTransportCredentials(insecure.NewCredentials()))
		},
		ReportInterval:      *reportIntervalFlag,
		HealthCheckInterval: *healthCheckIntervalFlag,
		ValidateCRCInterval: *crcCheckIntervalFlag,
	}

	nodeService, err := node.NewService(serviceOpts)
	if err != nil {
		log.WithError(err).Fatal("Failed to create service")
	}

	nodeServer, err := node.NewServer(node.ServerOpts{
		Logger:  log,
		Service: nodeService,
	})
	if err != nil {
		log.WithError(err).Fatal("Failed to create server")
	}

	listener, err := net.Listen("tcp", *hostFlag)
	if err != nil {
		log.WithError(err).Fatal("Failed to listen")
	}

	grpcServer := grpc.NewServer()
	proto.RegisterNodeServer(grpcServer, nodeServer)

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
	err = db.AutoMigrate(node.BlockInfo{})
	if err != nil {
		return nil, fmt.Errorf("failed to auto migrate: %w", err)
	}

	return db, nil
}
