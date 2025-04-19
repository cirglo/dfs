package client

import (
	"fmt"
	"github.com/cirglo.com/dfs/pkg/proto"
)

type SequenceNumber uint64
type BlockSize uint32
type BlockID string
type HostName string

type DFSClient interface {
	CreateFile(path string) error
	CreateDirectory(path string) error
	DeleteFile(path string) error
	DeleteDirectory(path string) error
	ListFiles(path string) ([]string, error)
	WriteFile(path string, data []byte) (SequenceNumber, error)
	GetSequences(path string) (map[SequenceNumber]BlockSize, error)
	ReadSequence(path string, sequence SequenceNumber) ([]byte, error)
}

type NameClientFactory func(host string) (proto.NameClient, error)
type NodeClientFactory func(host string) (proto.NodeClient, error)

type Opts struct {
	NameNode          string
	NameClientFactory NameClientFactory
	NodeClientFactory NodeClientFactory
}

func (o *Opts) Validate() error {
	if o.NameNode == "" {
		return fmt.Errorf("Name Node is required")
	}

	if o.NameClientFactory == nil {
		return fmt.Errorf("Name Client factory is required")
	}

	if o.NodeClientFactory == nil {
		return fmt.Errorf("Name Client factory is required")
	}

	return nil
}

type client struct {
	Opts           Opts
	NameClient     proto.NameClient
	NodeClients    map[HostName]proto.NodeClient
	BlockLocations map[BlockID][]HostName
}

var _ DFSClient = &client{}

func New(opts Opts) (DFSClient, error) {
	err := opts.Validate()
	if err != nil {
		return nil, fmt.Errorf("Opts failed validation: %w", err)
	}

	nameClient, err := opts.NameClientFactory(opts.NameNode)
	if err != nil {
		return nil, fmt.Errorf("Failed to create nameClient at '%s': %w", opts.NameNode, err)
	}

	return &client{
		Opts:           opts,
		NameClient:     nameClient,
		NodeClients:    map[HostName]proto.NodeClient{},
		BlockLocations: map[BlockID][]HostName{},
	}, nil
}

func (c client) CreateFile(path string) error {
	//TODO implement me
	panic("implement me")
}

func (c client) CreateDirectory(path string) error {
	//TODO implement me
	panic("implement me")
}

func (c client) DeleteFile(path string) error {
	//TODO implement me
	panic("implement me")
}

func (c client) DeleteDirectory(path string) error {
	//TODO implement me
	panic("implement me")
}

func (c client) ListFiles(path string) ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func (c client) WriteFile(path string, data []byte) (SequenceNumber, error) {
	//TODO implement me
	panic("implement me")
}

func (c client) GetSequences(path string) (map[SequenceNumber]BlockSize, error) {
	//TODO implement me
	panic("implement me")
}

func (c client) ReadSequence(path string, sequence SequenceNumber) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}
