package auth

import (
	"context"
	"fmt"
	"github.com/cirglo.com/dfs/pkg/proto"
	"github.com/cirglo.com/dfs/pkg/security"
)

type server struct {
	proto.UnimplementedAuthServer
	security.Service
}

var _ proto.AuthServer = (*server)(nil)

func (a server) Login(ctx context.Context, request *proto.LoginRequest) (*proto.LoginResponse, error) {
	token, err := a.Service.AuthenticateUser(request.GetUser(), request.GetHashedPassword())
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	return &proto.LoginResponse{Token: token}, nil
}

func (a server) Logout(ctx context.Context, request *proto.LogoutRequest) (*proto.LogoutResponse, error) {
	err := a.Service.Logout(request.GetToken())
	return &proto.LogoutResponse{}, err
}
