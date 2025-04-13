package name

import (
	"context"
	"fmt"
	"github.com/cirglo.com/dfs/pkg/proto"
)

type InternalServer struct {
	proto.UnimplementedNameInternalServer
	FileService FileService
}

func indexReport(report *proto.BlockInfoReport) map[string][]BlockInfo {
	index := map[string][]BlockInfo{}

	for _, blockInfoItem := range report.GetBlockInfos() {
		path := blockInfoItem.GetPath()
		if _, ok := index[path]; !ok {
			index[path] = []BlockInfo{}
		}
		blockInfo := BlockInfo{
			ID:       blockInfoItem.GetBlockId(),
			Sequence: blockInfoItem.GetSequence(),
			Length:   blockInfoItem.GetLength(),
			CRC:      blockInfoItem.GetCrc(),
			Locations: []Location{{
				Location: report.Host,
			}},
		}

		index[path] = append(index[path], blockInfo)
	}

	return index
}

func (i InternalServer) ReportExistingBlocks(ctx context.Context, report *proto.BlockInfoReport) (*proto.BlockInfoReportResponse, error) {
	index := indexReport(report)
	for path, blocks := range index {
		err := i.FileService.UpsertBlockInfos(NewRootPrincipal(), path, blocks)
		if err != nil {
			return nil, fmt.Errorf("failed to upsert block info: %w", err)
		}
	}

	return &proto.BlockInfoReportResponse{}, nil
}

func (i InternalServer) NotifyBlocksAdded(ctx context.Context, report *proto.BlockInfoReport) (*proto.BlockInfoReportResponse, error) {
	index := indexReport(report)
	for path, blocks := range index {
		err := i.FileService.UpsertBlockInfos(NewRootPrincipal(), path, blocks)
		if err != nil {
			return nil, fmt.Errorf("failed to upsert block info: %w", err)
		}
	}

	return &proto.BlockInfoReportResponse{}, nil
}

func (i InternalServer) NotifyBlocksRemoved(ctx context.Context, report *proto.BlockInfoReport) (*proto.BlockInfoReportResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (i InternalServer) mustEmbedUnimplementedNameInternalServer() {
	//TODO implement me
	panic("implement me")
}
