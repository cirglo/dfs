package proto

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ConnectionFactory func(host string) (*grpc.ClientConn, error)

func NewInsecureConnectionFactory() ConnectionFactory {
	return func(host string) (*grpc.ClientConn, error) {
		return grpc.NewClient(host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
}
