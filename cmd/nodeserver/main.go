package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/cirglo.com/dfs/pkg/node"
	"github.com/cirglo.com/dfs/pkg/proto"
	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	logLevelFlag := flag.String("log-level", "info", "Log Level")
	idFlag := flag.String("id", "default-id", "Node ID")
	locationFlag := flag.String("location", "/", "Node Location")
	dirFlag := flag.String("dir", "./", "Node Directory")
	portFlag := flag.Int("port", 50051, "Node Port")
	healthCheckIntervalFlag := flag.Duration("health-check-interval", 1*time.Minute, "Health Check Interval")
	crcCheckIntervalFlag := flag.Duration("crc-check-interval", 24*time.Hour, "CRC Check Interval")
	leaseDurationFlag := flag.Duration("lease-duration", 2*time.Minute, "Lease Duration")
	etcdIntervalFlag := flag.Duration("etcd-interval", 1*time.Minute, "ETCD Interval")
	etcdTimeoutFlag := flag.Duration("etcd-timeout", 5*time.Second, "ETCD Timeout")
	etcdEndpointsFlag := flag.String("etcd-endpoints", "localhost:2379", "ETCD Endpoints")
	etcdUsernameFlag := flag.String("etcd-username", "", "ETCD Username")
	etcPasswordFlag := flag.String("etcd-password", "", "ETCD Password")

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

	client, err := clientv3.New(clientv3.Config{
		Endpoints: strings.Split(*etcdEndpointsFlag, ","),
		Username:  *etcdUsernameFlag,
		Password:  *etcPasswordFlag,
	})
	if err != nil {
		log.WithError(err).Fatal("Failed to create etcd client")
	}

	etcdOpts := node.EtcdOpts{
		ID:            *idFlag,
		Host:          "localhost:2379",
		LeaseDuration: *leaseDurationFlag,
		ContextFactory: func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), *etcdTimeoutFlag)
		},
		Client: client,
	}

	etcd, err := node.NewEtcd(etcdOpts)
	if err != nil {
		log.WithError(err).Fatal("Failed to create etcd client")
	}

	go func() {
		ticker := time.NewTicker(*etcdIntervalFlag)
		defer ticker.Stop()
		for range ticker.C {
			err := etcd.Report()
			if err != nil {
				log.WithError(err).Fatal("Failed to report to etcd")
			} else {
				log.Info("Reported to etcd successfully")
			}
		}
	}()

	serviceOpts := node.ServiceOpts{
		Logger:              log,
		ID:                  *idFlag,
		Location:            *locationFlag,
		Dir:                 dir,
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

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", *portFlag))
	if err != nil {
		log.WithError(err).WithField("port", *portFlag).Fatal("Failed to listen")
	}

	grpcServer := grpc.NewServer()
	proto.RegisterNodeServer(grpcServer, nodeServer)

	log.WithField("port", *portFlag).Infof("Starting server")
	if err := grpcServer.Serve(listener); err != nil {
		log.WithError(err).Fatal("Failed to serve gRPC server")
	}
}
