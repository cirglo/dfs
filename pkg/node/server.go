package node

import (
	"context"
	"fmt"
	"github.com/cirglo.com/dfs/pkg/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type ServerOpts struct {
	Logger                  *logrus.Logger
	Service                 Service
	ClientConnectionFactory func(destination string) (*grpc.ClientConn, error)
}

func (s ServerOpts) Validate() error {
	if s.Logger == nil {
		return fmt.Errorf("no logger provided")
	}
	if s.Service == nil {
		return fmt.Errorf("no service provided")
	}
	if s.ClientConnectionFactory == nil {
		return fmt.Errorf("no client connection factory provided")
	}
	return nil
}

type server struct {
	opts ServerOpts
	proto.UnimplementedNodeServer
}

var _ proto.NodeServer = &server{}

func NewServer(opts ServerOpts) (proto.NodeServer, error) {
	err := opts.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}
	return &server{
		opts: opts,
	}, nil
}

func (s *server) GetBlockInfos(ctx context.Context, _ *proto.GetBlockInfosRequest) (*proto.GetBlockInfosResponse, error) {
	bis, err := s.opts.Service.GetBlocks()
	if err != nil {
		return nil, err
	}

	blockInfos := []*proto.BlockInfo{}

	for _, bi := range bis {
		blockInfo := &proto.BlockInfo{
			BlockId:  bi.ID,
			Crc:      bi.CRC,
			Sequence: bi.Sequence,
			Length:   bi.Length,
		}

		blockInfos = append(blockInfos, blockInfo)
	}

	return &proto.GetBlockInfosResponse{BlockInfos: blockInfos}, nil
}

func (s *server) GetBlockInfo(ctx context.Context, request *proto.GetBlockInfoRequest) (*proto.GetBlockInfoResponse, error) {
	bis, err := s.opts.Service.GetBlocks()
	if err != nil {
		return nil, err
	}

	for _, bi := range bis {
		if bi.ID == request.GetId() {
			return &proto.GetBlockInfoResponse{BlockInfo: &proto.BlockInfo{
				BlockId:  bi.ID,
				Crc:      bi.CRC,
				Sequence: bi.Sequence,
				Length:   bi.Length,
				Path:     bi.Path,
			}}, nil
		}
	}

	return nil, fmt.Errorf("block %s not found", request.GetId())
}

func (s *server) GetBlock(ctx context.Context, request *proto.GetBlockRequest) (*proto.GetBlockResponse, error) {
	b, bi, err := s.opts.Service.ReadBlock(request.GetId())
	if err != nil {
		return nil, err
	}

	return &proto.GetBlockResponse{
		Data: b,
		BlockInfo: &proto.BlockInfo{
			BlockId:  bi.ID,
			Crc:      bi.CRC,
			Sequence: bi.Sequence,
			Length:   bi.Length,
			Path:     bi.Path,
		}}, nil
}

func (s *server) WriteBlock(ctx context.Context, request *proto.WriteBlockRequest) (*proto.WriteBlockResponse, error) {
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
		Crc:      request.GetBlockInfo().GetCrc(),
		Sequence: request.GetBlockInfo().GetSequence(),
		Length:   request.GetBlockInfo().GetLength(),
		Path:     request.GetBlockInfo().GetPath(),
	},
	}, nil
}

func (s *server) DeleteBlock(ctx context.Context, request *proto.DeleteBlockRequest) (*proto.DeleteBlockResponse, error) {
	err := s.opts.Service.DeleteBlock(request.GetId())
	if err != nil {
		return nil, err
	}

	return &proto.DeleteBlockResponse{}, nil
}

func (s *server) CopyBlock(ctx context.Context, request *proto.CopyBlockRequest) (*proto.CopyBlockResponse, error) {
	data, blockInfo, err := s.opts.Service.ReadBlock(request.GetId())
	if err != nil {
		return nil, fmt.Errorf("failed to read data for block id %s : %w", blockInfo.ID, err)
	}

	conn, err := s.opts.ClientConnectionFactory(request.GetDestination())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to destination node: %w", err)
	}
	defer conn.Close()
	client := proto.NewNodeClient(conn)
	_, err = client.WriteBlock(ctx, &proto.WriteBlockRequest{
		BlockInfo: &proto.BlockInfo{
			BlockId:  blockInfo.ID,
			Crc:      blockInfo.CRC,
			Sequence: blockInfo.Sequence,
			Length:   blockInfo.Length,
			Path:     blockInfo.Path,
		},
		Data: data})
	if err != nil {
		return nil, fmt.Errorf("failed to write data for block id %s : %w", blockInfo.ID, err)
	}

	return &proto.CopyBlockResponse{}, nil
}

func (s *server) mustEmbedUnimplementedNodeServer() {
	//TODO implement me
	panic("implement me")
}
