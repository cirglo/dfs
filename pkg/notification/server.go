package notification

import (
	"context"
	"github.com/cirglo.com/dfs/pkg/file"
	"github.com/cirglo.com/dfs/pkg/healing"
	"github.com/cirglo.com/dfs/pkg/proto"
	"time"
)

type Server struct {
	proto.UnimplementedNotificationServer
	FileService    file.Service
	HealingService healing.Service
}

var _ proto.NotificationServer = (*Server)(nil)

func (n Server) NotifyBlockPresent(ctx context.Context, request *proto.NotifyBlockPresentRequest) (*proto.NotifyBlockPresentResponse, error) {
	n.HealingService.NotifyNodeAlive(request.Host, time.Now())
	err := n.FileService.NotifyBlockPresent(request)
	return &proto.NotifyBlockPresentResponse{}, err
}

func (n Server) NotifyBlockAdded(ctx context.Context, request *proto.NotifyBlockAddedRequest) (*proto.NotifyBlockAddedResponse, error) {
	n.HealingService.NotifyNodeAlive(request.Host, time.Now())
	err := n.FileService.NotifyBlockAdded(request)
	return &proto.NotifyBlockAddedResponse{}, err
}

func (n Server) NotifyBlockRemoved(ctx context.Context, request *proto.NotifyBlockRemovedRequest) (*proto.NotifyBlockRemovedResponse, error) {
	n.HealingService.NotifyNodeAlive(request.Host, time.Now())
	err := n.FileService.NotifyBlockRemoved(request)
	return &proto.NotifyBlockRemovedResponse{}, err
}
