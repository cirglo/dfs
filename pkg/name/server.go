package name

import (
	"context"
	"fmt"
	"github.com/cirglo.com/dfs/pkg/proto"
	"github.com/sirupsen/logrus"
)

type ServerOpts struct {
	Logger          *logrus.Logger
	SecurityService SecurityService
	FileService     FileService
}

type Server struct {
	proto.UnimplementedNameServer
	Opts ServerOpts
}

func (s Server) Login(ctx context.Context, request *proto.LoginRequest) (*proto.LoginResponse, error) {
	token, err := s.Opts.SecurityService.AuthenticateUser(request.GetUser(), request.GetHashedPassword())
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	return &proto.LoginResponse{
		Token: token,
	}, nil
}

func (s Server) Logout(ctx context.Context, request *proto.LogoutRequest) (*proto.LogoutResponse, error) {
	err := s.Opts.SecurityService.Logout(request.GetToken())
	if err != nil {
		return nil, fmt.Errorf("logout failed: %w", err)
	}

	return &proto.LogoutResponse{}, nil
}

func convertProtoPermission(permission *proto.Permission) Permission {
	return Permission{
		Read:  permission.GetRead(),
		Write: permission.GetWrite(),
	}
}

func convertProtoPermissions(permissions *proto.Permissions) Permissions {
	return Permissions{
		Owner:           permissions.GetOwner(),
		Group:           permissions.GetGroup(),
		OwnerPermission: convertProtoPermission(permissions.GetOwnerPermission()),
		GroupPermission: convertProtoPermission(permissions.GetGroupPermission()),
		OtherPermission: convertProtoPermission(permissions.GetOtherPermission()),
	}
}

func convertToProtoPermission(permission Permission) *proto.Permission {
	return &proto.Permission{
		Read:  permission.Read,
		Write: permission.Write,
	}
}

func convertToProtoPermissions(permissions Permissions) *proto.Permissions {
	return &proto.Permissions{
		Owner:           permissions.Owner,
		Group:           permissions.Group,
		OwnerPermission: convertToProtoPermission(permissions.OwnerPermission),
		GroupPermission: convertToProtoPermission(permissions.GroupPermission),
		OtherPermission: convertToProtoPermission(permissions.OtherPermission),
	}
}

func (s Server) CreateFile(ctx context.Context, request *proto.CreateFileRequest) (*proto.CreateFileResponse, error) {
	user, err := s.Opts.SecurityService.LookupUserByToken(request.GetToken())
	if err != nil {
		return nil, fmt.Errorf("failed to lookup user: %w", err)
	}
	principal := NewPrincipal(user)
	permissions := convertProtoPermissions(request.GetPermissions())
	_, err = s.Opts.FileService.CreateFile(principal, request.GetPath(), permissions)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	return &proto.CreateFileResponse{}, nil
}

func (s Server) CreateDir(ctx context.Context, request *proto.CreateDirRequest) (*proto.CreateDirResponse, error) {
	user, err := s.Opts.SecurityService.LookupUserByToken(request.GetToken())
	if err != nil {
		return nil, fmt.Errorf("failed to lookup user: %w", err)
	}
	principal := NewPrincipal(user)
	permissions := convertProtoPermissions(request.GetPermissions())
	_, err = s.Opts.FileService.CreateDir(principal, request.GetPath(), permissions)
	if err != nil {
		return nil, fmt.Errorf("failed to create dir: %w", err)
	}
	return &proto.CreateDirResponse{}, nil
}

func (s Server) DeleteFile(ctx context.Context, request *proto.DeleteFileRequest) (*proto.DeleteFileResponse, error) {
	user, err := s.Opts.SecurityService.LookupUserByToken(request.GetToken())
	if err != nil {
		return nil, fmt.Errorf("failed to lookup user: %w", err)
	}
	principal := NewPrincipal(user)
	err = s.Opts.FileService.DeleteFile(principal, request.GetPath())
	if err != nil {
		return nil, fmt.Errorf("failed to delete file: %w", err)
	}
	return &proto.DeleteFileResponse{}, nil
}

func (s Server) DeleteDir(ctx context.Context, request *proto.DeleteDirRequest) (*proto.DeleteDirResponse, error) {
	user, err := s.Opts.SecurityService.LookupUserByToken(request.GetToken())
	if err != nil {
		return nil, fmt.Errorf("failed to lookup user: %w", err)
	}
	principal := NewPrincipal(user)
	err = s.Opts.FileService.DeleteDir(principal, request.GetPath())
	if err != nil {
		return nil, fmt.Errorf("failed to delete dir: %w", err)
	}
	return &proto.DeleteDirResponse{}, nil
}

func convertToProtoDirEntryFile(fileInfo FileInfo, parentDir string) *proto.DirEntry {
	path := fmt.Sprintf("%s/%s", parentDir, fileInfo.Name)
	return &proto.DirEntry{
		Path:        path,
		IsDir:       false,
		Permissions: convertToProtoPermissions(fileInfo.Permissions),
		CreatedAt:   fileInfo.CreatedAt.Unix(),
		ModifiedAt:  fileInfo.UpdatedAt.Unix(),
		AccessedAt:  fileInfo.UpdatedAt.Unix(),
	}
}

func (s Server) List(ctx context.Context, request *proto.ListRequest) (*proto.ListResponse, error) {
	user, err := s.Opts.SecurityService.LookupUserByToken(request.GetToken())
	if err != nil {
		return nil, fmt.Errorf("failed to lookup user: %w", err)
	}
	principal := NewPrincipal(user)

	fileInfos, err := s.Opts.FileService.List(principal, request.GetPath())
	if err != nil {
		return nil, fmt.Errorf("failed to list path '%s': %w", request.GetPath(), err)
	}

	var entries []*proto.DirEntry

	for _, fileInfo := range fileInfos {
		entries = append(entries, convertToProtoDirEntryFile(fileInfo, request.GetPath()))
	}

	return &proto.ListResponse{
		Path:    request.GetPath(),
		Entries: entries,
	}, nil
}

func convertToProtoStatBlockInfo(blockInfo BlockInfo) *proto.StatBlockInfo {
	return &proto.StatBlockInfo{
		BlockId:  blockInfo.ID,
		Crc:      blockInfo.CRC,
		Sequence: blockInfo.Sequence,
		Length:   blockInfo.Length,
	}
}

func (s Server) Stat(ctx context.Context, request *proto.StatRequest) (*proto.StatResponse, error) {
	user, err := s.Opts.SecurityService.LookupUserByToken(request.GetToken())
	if err != nil {
		return nil, fmt.Errorf("failed to lookup user: %w", err)
	}
	principal := NewPrincipal(user)
	fileInfo, err := s.Opts.FileService.Stat(principal, request.GetPath())
	if err != nil {
		return nil, fmt.Errorf("failed to stat file '%s': %w", request.GetPath(), err)
	}

	blockInfos, err := s.Opts.FileService.GetBlockInfos(principal, request.GetPath())
	if err != nil {
		return nil, fmt.Errorf("failed to get block infos: %w", err)
	}

	var protoBlockInfos []*proto.StatBlockInfo

	for _, blockInfo := range blockInfos {
		protoBlockInfos = append(protoBlockInfos, convertToProtoStatBlockInfo(blockInfo))
	}

	return &proto.StatResponse{
		Path:       request.GetPath(),
		Entry:      convertToProtoDirEntryFile(fileInfo, request.GetPath()),
		BlockInfos: protoBlockInfos,
	}, nil
}
