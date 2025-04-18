package main

import (
	"flag"
	"fmt"
	"github.com/cirglo.com/dfs/pkg/node"
	"github.com/cirglo.com/dfs/pkg/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net"
	"os"
	"time"
)

func main() {
	logLevelFlag := flag.String("log-level", "info", "Log Level")
	nameNodeFlag := flag.String("name-node", "localhost:53035", "Name Node Address")
	dirFlag := flag.String("dir", "./", "Node Directory")
	dsnFlag := flag.String("dsn", "nodeserver.db", "Data Source Name (DSN) for the database")
	hostFlag := flag.String("host", "localhost:55055", "Node Host")
	reportIntervalFlag := flag.Duration("report-interval", 10*time.Minute, "Report Interval")
	healthCheckIntervalFlag := flag.Duration("health-check-interval", 1*time.Hour, "Health Check Interval")
	crcCheckIntervalFlag := flag.Duration("crc-check-interval", 24*time.Hour, "CRC Check Interval")

	flag.Parse()

	log := logrus.New()
	logLevel, err := logrus.ParseLevel(*logLevelFlag)
	if err != nil {
		log.WithError(err).WithField("level", *logLevelFlag).Fatal("Invalid log level")
	}

	log.SetLevel(logLevel)

	log.WithField("dir-path", *dirFlag).Info("Checking directory")
	_, err = os.Stat(*dirFlag)
	if err != nil {
		log.WithError(err).WithField("dir", *dirFlag).Fatal("Directory does not exist")
	}

	log.WithField("dns", *dsnFlag).Info("Opening database")
	db, err := createDB(sqlite.Open(*dsnFlag))
	if err != nil {
		log.WithError(err).Fatal("Failed to create database")
	}

	connectionFactory := proto.NewInsecureConnectionFactory()

	log.WithField("name-node", *nameNodeFlag).Info("Connecting to name node")
	conn, err := connectionFactory.CreateConnection(*nameNodeFlag)
	if err != nil {
		log.WithError(err).Fatal("Failed to connect to name node")
	}
	client := proto.NewNotificationClient(conn)

	serviceOpts := node.BlockServiceOpts{
		Logger:             log,
		Host:               *hostFlag,
		DB:                 db,
		Dir:                *dirFlag,
		NotificationClient: client,
	}

	log.Info("Creating block service")
	blockService, err := node.NewBlockService(serviceOpts)
	if err != nil {
		log.WithError(err).Fatal("Failed to create block service")
	}

	log.Info("Reporting to name node")
	err = blockService.Report()
	if err != nil {
		log.WithError(err).Fatal("Failed to report to name node")
	}

	log.Info("Creating server")
	nodeServer, err := node.NewServer(node.ServerOpts{
		Logger:            log,
		BlockService:      blockService,
		ConnectionFactory: connectionFactory,
	})
	if err != nil {
		log.WithError(err).Fatal("Failed to create server")
	}

	log.WithField("host", *hostFlag).Info("Creating Network listener")
	listener, err := net.Listen("tcp", *hostFlag)
	if err != nil {
		log.WithError(err).Fatal("Failed to listen")
	}

	grpcServer := grpc.NewServer()
	proto.RegisterNodeServer(grpcServer, nodeServer)

	log.Info("Starting grpc server")
	if err := grpcServer.Serve(listener); err != nil {
		log.WithError(err).Fatal("Failed to serve gRPC server")
	}

	go func() {
		ticker := time.NewTicker(*reportIntervalFlag)
		for range ticker.C {
			log.Info("Report to name node")
			err := blockService.Report()
			if err != nil {
				log.WithError(err).Fatal("report to name node failed")
			}
			log.Info("Reported to name node")
		}
	}()

	go func() {
		ticker := time.NewTicker(*healthCheckIntervalFlag)
		for range ticker.C {
			log.Info("Performing health check")
			err := blockService.HealthCheck()
			if err != nil {
				log.WithError(err).Fatal("health check failed")
			}
			log.Info("Health check done")
		}
	}()

	go func() {
		ticker := time.NewTicker(*crcCheckIntervalFlag)
		for range ticker.C {
			log.Info("Validating CRC")
			err := blockService.ValidateCRC()
			if err != nil {
				log.WithError(err).Fatal("validate CRC failed")
			}
			log.Info("Finished validating CRC")
		}
	}()

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
