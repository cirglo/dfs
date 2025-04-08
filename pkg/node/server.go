package node

import (
	"context"
	"fmt"
	"github.com/cirglo.com/dfs/pkg/proto"
	"github.com/sirupsen/logrus"
)

type ServerOpts struct {
	Logger  *logrus.Logger
	Service Service
}

type server struct {
	opts ServerOpts
	proto.UnimplementedNodeServer
}

var _ proto.NodeServer = &server{}

func NewServer(opts ServerOpts) (proto.NodeServer, error) {
	if opts.Logger == nil {
		return nil, fmt.Errorf("no logger provided")
	}
	if opts.Service == nil {
		return nil, fmt.Errorf("no service provided")
	}
	return &server{
		opts: opts,
	}, nil
}

func (s server) GetBlockIds(ctx context.Context, request *proto.GetBlockIdsRequest) (*proto.GetBlockIdsResponse, error) {
	bids, err := s.opts.Service.GetBlockIds()
	if err != nil {
		return nil, err
	}

	return &proto.GetBlockIdsResponse{Ids: bids}, nil

}

func (s server) GetBlockInfo(ctx context.Context, request *proto.GetBlockInfoRequest) (*proto.GetBlockInfoResponse, error) {
	bis, err := s.opts.Service.GetBlocks()
	if err != nil {
		return nil, err
	}

	for _, bi := range bis {
		if bi.ID == request.GetId() {
			return &proto.GetBlockInfoResponse{BlockInfo: &proto.BlockInfo{
				BlockId:  bi.ID,
				FileId:   "",
				Crc:      bi.CRC,
				Sequence: bi.Sequence,
				Length:   bi.Length,
				Path:     bi.Path,
			}}, nil
		}
	}

	return nil, fmt.Errorf("block %s not found", request.GetId())
}

func (s server) GetBlock(ctx context.Context, request *proto.GetBlockRequest) (*proto.GetBlockResponse, error) {
	b, bi, err := s.opts.Service.ReadBlock(request.GetId())
	if err != nil {
		return nil, err
	}

	return &proto.GetBlockResponse{
		Data: b,
		BlockInfo: &proto.BlockInfo{
			BlockId:  bi.ID,
			FileId:   "",
			Crc:      bi.CRC,
			Sequence: bi.Sequence,
			Length:   bi.Length,
			Path:     bi.Path,
		}}, nil
}

func (s server) WriteBlock(ctx context.Context, request *proto.WriteBlockRequest) (*proto.WriteBlockResponse, error) {
	err := s.opts.Service.WriteBlock(BlockInfo{
		ID:       request.GetBlockInfo().GetBlockId(),
		Sequence: request.GetBlockInfo().GetSequence(),
		Length:   request.GetBlockInfo().GetLength(),
		Path:     request.GetBlockInfo().GetPath(),
		CRC:      request.GetBlockInfo().GetCrc(),
	},
		request.Data)

	if err != nil {
		return nil, err
	}

	return &proto.WriteBlockResponse{BlockInfo: &proto.BlockInfo{
		BlockId:  request.GetBlockInfo().GetBlockId(),
		FileId:   request.GetBlockInfo().GetFileId(),
		Crc:      request.GetBlockInfo().GetCrc(),
		Sequence: request.GetBlockInfo().GetSequence(),
		Length:   request.GetBlockInfo().GetLength(),
		Path:     request.GetBlockInfo().GetPath(),
	},
	}, nil
}

func (s server) DeleteBlock(ctx context.Context, request *proto.DeleteBlockRequest) (*proto.DeleteBlockResponse, error) {
	err := s.opts.Service.DeleteBlock(request.GetId())
	if err != nil {
		return nil, err
	}

	return &proto.DeleteBlockResponse{Id: request.GetId()}, nil
}

func (s server) CopyBlock(ctx context.Context, request *proto.CopyBlockRequest) (*proto.CopyBlockResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s server) mustEmbedUnimplementedNodeServer() {
	//TODO implement me
	panic("implement me")
}
