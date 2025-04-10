package name

import (
	"context"
	"github.com/cirglo.com/dfs/pkg/proto"
	"github.com/sirupsen/logrus"
)

type ServerOpts struct {
	Logger      *logrus.Logger
	FileService FileService
}

type Server struct {
}

func (s Server) Login(ctx context.Context, request *proto.LoginRequest) (*proto.LoginResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) Logout(ctx context.Context, request *proto.LogoutRequest) (*proto.LogoutResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) CreateFile(ctx context.Context, request *proto.CreateFileRequest) (*proto.CreateFileResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) CreateDir(ctx context.Context, request *proto.CreateDirRequest) (*proto.CreateDirResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) DeleteFile(ctx context.Context, request *proto.DeleteFileRequest) (*proto.DeleteFileResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) DeleteDir(ctx context.Context, request *proto.DeleteDirRequest) (*proto.DeleteDirResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) ListDir(ctx context.Context, request *proto.ListDirRequest) (*proto.ListDirResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) StatFile(ctx context.Context, request *proto.StatFileRequest) (*proto.StatFileResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) OpenFile(ctx context.Context, request *proto.OpenFileRequest) (*proto.OpenFileResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) CloseFile(ctx context.Context, request *proto.CloseFileRequest) (*proto.CloseFileResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) PrepareWrite(ctx context.Context, request *proto.PrepareWriteRequest) (*proto.PrepareWriteResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) mustEmbedUnimplementedNameServer() {
	//TODO implement me
	panic("implement me")
}

var _ proto.NameServer = Server{}

func NewServer(opts ServerOpts) *Server {
	return &Server{}
}
