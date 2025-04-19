package healing

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirglo.com/dfs/pkg/file"
	"github.com/cirglo.com/dfs/pkg/proto"
	"github.com/sirupsen/logrus"
	"math/rand"
	"slices"
	"sync"
	"time"
)

type Opts struct {
	Logger            *logrus.Logger
	NumReplicas       uint
	FileService       file.Service
	NodeExpiration    time.Duration
	ConnectionFactory proto.ConnectionFactory
}

func (o *Opts) Validate() error {
	if o.Logger == nil {
		return fmt.Errorf("logger is required")
	}

	if o.NumReplicas >= 255 {
		return fmt.Errorf("number of replicas must be less than 256")
	}

	if o.NumReplicas == 0 {
		return fmt.Errorf("num replicas is required")
	}

	if o.FileService == nil {
		return fmt.Errorf("fileService is required")
	}

	if o.ConnectionFactory == nil {
		return fmt.Errorf("connection factory is required")
	}

	return nil
}

type Service interface {
	NotifyNodeAlive(host string, at time.Time)
	Heal(since time.Time) error
}

type service struct {
	Opts  Opts
	Nodes map[string]time.Time
	Lock  sync.RWMutex
}

var _ Service = &service{}

func NewService(opts Opts) (Service, error) {
	err := opts.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	return &service{
		Opts:  opts,
		Nodes: map[string]time.Time{},
		Lock:  sync.RWMutex{},
	}, nil
}

func (s *service) NotifyNodeAlive(host string, at time.Time) {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	s.Nodes[host] = at
}

func (s *service) Heal(since time.Time) error {
	removedHosts := s.removeExpiredNodes(since)
	var allErrors []error
	for _, host := range removedHosts {
		s.Opts.Logger.WithField("host", host).Info("Removing expired node")
		err := s.Opts.FileService.NodeRemoved(host)
		allErrors = append(allErrors, err)
	}

	blockInfos, err := s.Opts.FileService.GetAllBlockInfos()
	if err != nil {
		return fmt.Errorf("could not get block infos: %w", err)
	}

	currentLocations := map[string][]string{}

	for _, blockInfo := range blockInfos {
		id := blockInfo.ID
		currentLocations[id] = []string{}

		for _, location := range blockInfo.Locations {
			host := location.Host
			currentLocations[id] = append(currentLocations[id], host)
		}
	}

	for id := range currentLocations {
		slices.Sort(currentLocations[id])
	}
	for _, blockInfo := range blockInfos {
		s.checkBlock(blockInfo, currentLocations[blockInfo.ID])
	}

	return errors.Join(allErrors...)
}

func (s *service) removeExpiredNodes(since time.Time) []string {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	var toRemove []string

	for host, at := range s.Nodes {
		expiration := at.Add(s.Opts.NodeExpiration)
		if expiration.Before(since) {
			toRemove = append(toRemove, host)
		}
	}

	for _, host := range toRemove {
		s.Opts.Logger.WithField("host", host).Info("node is dead")
		delete(s.Nodes, host)
	}

	return toRemove
}

func (s *service) checkBlock(blockInfo file.BlockInfo, currentLocations []string) {
	s.Lock.RLock()
	defer s.Lock.RUnlock()

	neededCount := int(s.Opts.NumReplicas) - len(blockInfo.Locations)

	if neededCount > 0 {
		s.Opts.Logger.WithFields(logrus.Fields{
			"block-id":                  blockInfo.ID,
			"mandatory-replicas-count":  s.Opts.NumReplicas,
			"replicas-count":            len(blockInfo.Locations),
			"needed-new-replicas-count": neededCount,
		}).Info("Block needs more replicas")
		destinations, found := s.findDestinations(currentLocations, neededCount)
		if found {
			for _, destination := range destinations {
				if len(currentLocations) == 0 {
					s.Opts.Logger.WithField("block-id", blockInfo.ID).Warn("No current locations available to select a source for block replication")
					continue
				}
				source := currentLocations[rand.Intn(len(currentLocations))]
				go s.copyBlock(blockInfo.ID, source, destination)
			}
		}
	}
}

func (s *service) findDestinations(currentLocations []string, count int) ([]string, bool) {
	var candidates []string

	for location := range s.Nodes {
		_, found := slices.BinarySearch(currentLocations, location)
		if !found {
			candidates = append(candidates, location)
		}
	}

	if len(candidates) < count {
		return nil, false
	}

	shuffle(candidates)

	return candidates[:count], true
}

func (s *service) copyBlock(blockId string, source string, dest string) {
	connection, err := s.Opts.ConnectionFactory.CreateConnection(source)
	if err != nil {
		s.Opts.Logger.WithError(err).WithField("host", dest).Error("could not create connection")
		return
	}
	defer connection.Close()

	client := proto.NewNodeClient(connection)

	s.Opts.Logger.WithFields(logrus.Fields{
		"source":      source,
		"destination": dest,
		"block-id":    blockId,
	}).Info("Copying block")
	_, err = client.CopyBlock(context.Background(), &proto.CopyBlockRequest{
		Id:          blockId,
		Destination: dest,
	})
	if err != nil {
		s.Opts.Logger.
			WithError(err).
			WithFields(logrus.Fields{
				"block-id":    blockId,
				"source":      source,
				"destination": dest,
			}).
			Error("unable to copy block")
	} else {
		s.Opts.Logger.WithFields(logrus.Fields{
			"source":      source,
			"destination": dest,
			"block-id":    blockId,
		}).Info("block copied")
	}
}

func shuffle(slice []string) {
	for i := range slice {
		j := rand.Intn(len(slice))
		slice[i], slice[j] = slice[j], slice[i]
	}
}
