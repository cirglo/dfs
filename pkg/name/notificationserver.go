package name

import (
	"context"
	"github.com/cirglo.com/dfs/pkg/proto"
	"time"
)

type NotificationServer struct {
	proto.UnimplementedNotificationServer
	FileService    FileService
	HealingService HealingService
}

var _ proto.NotificationServer = (*NotificationServer)(nil)

func (n NotificationServer) NotifyBlockPresent(ctx context.Context, request *proto.NotifyBlockPresentRequest) (*proto.NotifyBlockPresentResponse, error) {
	n.HealingService.NotifyNodeAlive(request.Host, time.Now())
	err := n.FileService.NotifyBlockPresent(request)
	return &proto.NotifyBlockPresentResponse{}, err
}

func (n NotificationServer) NotifyBlockAdded(ctx context.Context, request *proto.NotifyBlockAddedRequest) (*proto.NotifyBlockAddedRequest, error) {
	n.HealingService.NotifyNodeAlive(request.Host, time.Now())
	err := n.FileService.NotifyBlockAdded(request)
	return &proto.NotifyBlockAddedRequest{}, err
}

func (n NotificationServer) NotifyBlockRemoved(ctx context.Context, request *proto.NotifyBlockRemovedRequest) (*proto.NotifyBlockRemovedResponse, error) {
	n.HealingService.NotifyNodeAlive(request.Host, time.Now())
	err := n.FileService.NotifyBlockRemoved(request)
	return &proto.NotifyBlockRemovedResponse{}, err
}
