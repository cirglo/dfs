package node_test

import (
	"context"
	"testing"

	"github.com/cirglo.com/dfs/pkg/mocks"
	"github.com/cirglo.com/dfs/pkg/node"
	"github.com/cirglo.com/dfs/pkg/proto"
	"github.com/stretchr/testify/assert"
)

func createServer(t *testing.T, blockService *mocks.BlockService, connectionFactory *mocks.ConnectionFactory) proto.NodeServer {
	logger := createLogger(t)
	opts := node.ServerOpts{
		Logger:            logger,
		BlockService:      blockService,
		ConnectionFactory: connectionFactory,
	}
	server, err := node.NewServer(opts)
	assert.NoError(t, err)
	return server
}

func TestServer_GetBlockInfos(t *testing.T) {
	blockService := mocks.NewBlockService(t)
	connectionFactory := mocks.NewConnectionFactory(t)
	server := createServer(t, blockService, connectionFactory)

	blockService.On("GetBlocks").Return([]node.BlockInfo{
		{ID: "block1", CRC: 123, Sequence: 1, Length: 100},
	}, nil)

	resp, err := server.GetBlockInfos(context.Background(), &proto.GetBlockInfosRequest{})
	assert.NoError(t, err)
	assert.Len(t, resp.BlockInfos, 1)
	assert.Equal(t, "block1", resp.BlockInfos[0].BlockId)
}

func TestServer_GetBlockInfo(t *testing.T) {
	blockService := mocks.NewBlockService(t)
	connectionFactory := mocks.NewConnectionFactory(t)
	server := createServer(t, blockService, connectionFactory)

	blockService.On("GetBlocks").Return([]node.BlockInfo{
		{ID: "block1", CRC: 123, Sequence: 1, Length: 100, Path: "/path/to/block"},
	}, nil)

	resp, err := server.GetBlockInfo(context.Background(), &proto.GetBlockInfoRequest{Id: "block1"})
	assert.NoError(t, err)
	assert.Equal(t, "block1", resp.BlockInfo.BlockId)
	assert.Equal(t, "/path/to/block", resp.BlockInfo.Path)
}

func TestServer_GetBlock(t *testing.T) {
	blockService := mocks.NewBlockService(t)
	connectionFactory := mocks.NewConnectionFactory(t)
	server := createServer(t, blockService, connectionFactory)

	blockService.On("ReadBlock", "block1").Return([]byte("data"), node.BlockInfo{
		ID: "block1", CRC: 123, Sequence: 1, Length: 100, Path: "/path/to/block",
	}, nil)

	resp, err := server.GetBlock(context.Background(), &proto.GetBlockRequest{Id: "block1"})
	assert.NoError(t, err)
	assert.Equal(t, []uint8("data"), resp.Data)
	assert.Equal(t, "block1", resp.BlockInfo.BlockId)
}

func TestServer_WriteBlock(t *testing.T) {
	blockService := mocks.NewBlockService(t)
	connectionFactory := mocks.NewConnectionFactory(t)
	server := createServer(t, blockService, connectionFactory)

	blockService.On("WriteBlock", "block1", "/path/to/block", uint64(1), []byte("data")).Return(nil)

	_, err := server.WriteBlock(context.Background(), &proto.WriteBlockRequest{
		Id:       "block1",
		Path:     "/path/to/block",
		Sequence: 1,
		Data:     []byte("data"),
	})
	assert.NoError(t, err)
}

func TestServer_DeleteBlock(t *testing.T) {
	blockService := mocks.NewBlockService(t)
	connectionFactory := mocks.NewConnectionFactory(t)
	server := createServer(t, blockService, connectionFactory)

	blockService.On("DeleteBlock", "block1").Return(nil)

	_, err := server.DeleteBlock(context.Background(), &proto.DeleteBlockRequest{Id: "block1"})
	assert.NoError(t, err)
}
