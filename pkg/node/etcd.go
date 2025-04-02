package node

import (
	"context"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

const (
	nameFormat = "dfs/datanode/%s"
)

type Etcd interface {
	Report() error
}

type ContextFactory func() (context.Context, context.CancelFunc)

type EtcdOpts struct {
	Client         *clientv3.Client
	ID             string
	Host           string
	LeaseDuration  time.Duration
	ContextFactory ContextFactory
}

func NewEtcd(opts EtcdOpts) (Etcd, error) {
	if opts.Client == nil {
		return nil, fmt.Errorf("client cannot be nil")
	}
	if opts.ID == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}
	if opts.Host == "" {
		return nil, fmt.Errorf("host cannot be empty")
	}
	if opts.LeaseDuration <= 0 {
		return nil, fmt.Errorf("lease duration must be greater than 0")
	}
	if opts.ContextFactory == nil {
		return nil, fmt.Errorf("context factory cannot be nil")
	}

	return &etcd{opts: opts}, nil
}

type etcd struct {
	opts EtcdOpts
}

func (e *etcd) Report() error {
	ctx, cancel := e.opts.ContextFactory()
	defer cancel()
	ttl := e.opts.LeaseDuration.Seconds()
	gresp, err := e.opts.Client.Grant(ctx, int64(ttl))
	if err != nil {
		return fmt.Errorf("grant failed: %w", err)
	}

	name := fmt.Sprintf(nameFormat, e.opts.ID)
	_, err = e.opts.Client.Put(ctx, name, e.opts.Host, clientv3.WithLease(gresp.ID))
	if err != nil {
		return fmt.Errorf("put failed: %w", err)
	}

	return nil
}
