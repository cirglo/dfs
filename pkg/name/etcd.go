package name

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cirglo.com/dfs/pkg/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Etcd interface {
	Gather() ([]etcd.NodeInfo, error)
}

type ContextFactory func() (context.Context, context.CancelFunc)

type EtcdOpts struct {
	Client         *clientv3.Client
	ContextFactory ContextFactory
}

func NewEtcd(opts EtcdOpts) (Etcd, error) {
	if opts.Client == nil {
		return nil, fmt.Errorf("client cannot be nil")
	}
	if opts.ContextFactory == nil {
		return nil, fmt.Errorf("context factory cannot be nil")
	}

	return &etcdImpl{opts: opts}, nil
}

type etcdImpl struct {
	opts EtcdOpts
}

func (e *etcdImpl) Gather() ([]etcd.NodeInfo, error) {
	ctx, cancel := e.opts.ContextFactory()
	defer cancel()

	resp, err := e.opts.Client.Get(ctx, etcd.NameKeyPrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to get from etcd: %w", err)
	}

	result := []etcd.NodeInfo{}

	for _, kv := range resp.Kvs {
		nodeInfo := etcd.NodeInfo{}
		err := json.Unmarshal(kv.Value, &nodeInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal etcd value: %w", err)
		}
		result = append(result, nodeInfo)
	}

	return result, nil
}
