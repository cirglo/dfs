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
	//TODO implement me
	panic("implement me")
}

func (n NotificationServer) NotifyBlockAdded(ctx context.Context, request *proto.NotifyBlockAddedRequest) (*proto.NotifyBlockAddedRequest, error) {
	//TODO implement me
	panic("implement me")
}

func (n NotificationServer) NotifyBlockRemoved(ctx context.Context, request *proto.NotifyBlockRemovedRequest) (*proto.NotifyBlockRemovedResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (n NotificationServer) mustEmbedUnimplementedNotificationServer() {
	//TODO implement me
	panic("implement me")
}
