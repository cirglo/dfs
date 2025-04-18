package proto

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ConnectionFactory interface {
	CreateConnection(target string) (*grpc.ClientConn, error)
}

type insecureConnectionFactory struct {
}

var _ ConnectionFactory = &insecureConnectionFactory{}

func (i insecureConnectionFactory) CreateConnection(target string) (*grpc.ClientConn, error) {
	return grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
}

func NewInsecureConnectionFactory() ConnectionFactory {
	return &insecureConnectionFactory{}
}
