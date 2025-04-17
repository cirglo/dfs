package name

import (
	"context"
	"github.com/cirglo.com/dfs/pkg/proto"
)

type NotificationServer struct {
	proto.UnimplementedNotificationServer
	FileService FileService
}

var _ proto.NotificationServer = (*NotificationServer)(nil)

func (n NotificationServer) NotifyBlockPresent(ctx context.Context, request *proto.NotifyBlockPresentRequest) (*proto.NotifyBlockPresentResponse, error) {
	err := n.FileService.NotifyBlockPresent(request)
	return &proto.NotifyBlockPresentResponse{}, err
}

func (n NotificationServer) NotifyBlockAdded(ctx context.Context, request *proto.NotifyBlockAddedRequest) (*proto.NotifyBlockAddedRequest, error) {
	err := n.FileService.NotifyBlockAdded(request)
	return &proto.NotifyBlockAddedRequest{}, err
}

func (n NotificationServer) NotifyBlockRemoved(ctx context.Context, request *proto.NotifyBlockRemovedRequest) (*proto.NotifyBlockRemovedResponse, error) {
	err := n.FileService.NotifyBlockRemoved(request)
	return &proto.NotifyBlockRemovedResponse{}, err
}
