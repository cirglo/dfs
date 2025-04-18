// Code generated by mockery v2.53.3. DO NOT EDIT.

package mocks

import (
	context "context"

	proto "github.com/cirglo.com/dfs/pkg/proto"
	mock "github.com/stretchr/testify/mock"
)

// NodeServer is an autogenerated mock type for the NodeServer type
type NodeServer struct {
	mock.Mock
}

type NodeServer_Expecter struct {
	mock *mock.Mock
}

func (_m *NodeServer) EXPECT() *NodeServer_Expecter {
	return &NodeServer_Expecter{mock: &_m.Mock}
}

// CopyBlock provides a mock function with given fields: _a0, _a1
func (_m *NodeServer) CopyBlock(_a0 context.Context, _a1 *proto.CopyBlockRequest) (*proto.CopyBlockResponse, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for CopyBlock")
	}

	var r0 *proto.CopyBlockResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *proto.CopyBlockRequest) (*proto.CopyBlockResponse, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *proto.CopyBlockRequest) *proto.CopyBlockResponse); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*proto.CopyBlockResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *proto.CopyBlockRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NodeServer_CopyBlock_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CopyBlock'
type NodeServer_CopyBlock_Call struct {
	*mock.Call
}

// CopyBlock is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 *proto.CopyBlockRequest
func (_e *NodeServer_Expecter) CopyBlock(_a0 interface{}, _a1 interface{}) *NodeServer_CopyBlock_Call {
	return &NodeServer_CopyBlock_Call{Call: _e.mock.On("CopyBlock", _a0, _a1)}
}

func (_c *NodeServer_CopyBlock_Call) Run(run func(_a0 context.Context, _a1 *proto.CopyBlockRequest)) *NodeServer_CopyBlock_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*proto.CopyBlockRequest))
	})
	return _c
}

func (_c *NodeServer_CopyBlock_Call) Return(_a0 *proto.CopyBlockResponse, _a1 error) *NodeServer_CopyBlock_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *NodeServer_CopyBlock_Call) RunAndReturn(run func(context.Context, *proto.CopyBlockRequest) (*proto.CopyBlockResponse, error)) *NodeServer_CopyBlock_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteBlock provides a mock function with given fields: _a0, _a1
func (_m *NodeServer) DeleteBlock(_a0 context.Context, _a1 *proto.DeleteBlockRequest) (*proto.DeleteBlockResponse, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for DeleteBlock")
	}

	var r0 *proto.DeleteBlockResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *proto.DeleteBlockRequest) (*proto.DeleteBlockResponse, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *proto.DeleteBlockRequest) *proto.DeleteBlockResponse); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*proto.DeleteBlockResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *proto.DeleteBlockRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NodeServer_DeleteBlock_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteBlock'
type NodeServer_DeleteBlock_Call struct {
	*mock.Call
}

// DeleteBlock is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 *proto.DeleteBlockRequest
func (_e *NodeServer_Expecter) DeleteBlock(_a0 interface{}, _a1 interface{}) *NodeServer_DeleteBlock_Call {
	return &NodeServer_DeleteBlock_Call{Call: _e.mock.On("DeleteBlock", _a0, _a1)}
}

func (_c *NodeServer_DeleteBlock_Call) Run(run func(_a0 context.Context, _a1 *proto.DeleteBlockRequest)) *NodeServer_DeleteBlock_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*proto.DeleteBlockRequest))
	})
	return _c
}

func (_c *NodeServer_DeleteBlock_Call) Return(_a0 *proto.DeleteBlockResponse, _a1 error) *NodeServer_DeleteBlock_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *NodeServer_DeleteBlock_Call) RunAndReturn(run func(context.Context, *proto.DeleteBlockRequest) (*proto.DeleteBlockResponse, error)) *NodeServer_DeleteBlock_Call {
	_c.Call.Return(run)
	return _c
}

// GetBlock provides a mock function with given fields: _a0, _a1
func (_m *NodeServer) GetBlock(_a0 context.Context, _a1 *proto.GetBlockRequest) (*proto.GetBlockResponse, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for GetBlock")
	}

	var r0 *proto.GetBlockResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *proto.GetBlockRequest) (*proto.GetBlockResponse, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *proto.GetBlockRequest) *proto.GetBlockResponse); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*proto.GetBlockResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *proto.GetBlockRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NodeServer_GetBlock_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetBlock'
type NodeServer_GetBlock_Call struct {
	*mock.Call
}

// GetBlock is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 *proto.GetBlockRequest
func (_e *NodeServer_Expecter) GetBlock(_a0 interface{}, _a1 interface{}) *NodeServer_GetBlock_Call {
	return &NodeServer_GetBlock_Call{Call: _e.mock.On("GetBlock", _a0, _a1)}
}

func (_c *NodeServer_GetBlock_Call) Run(run func(_a0 context.Context, _a1 *proto.GetBlockRequest)) *NodeServer_GetBlock_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*proto.GetBlockRequest))
	})
	return _c
}

func (_c *NodeServer_GetBlock_Call) Return(_a0 *proto.GetBlockResponse, _a1 error) *NodeServer_GetBlock_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *NodeServer_GetBlock_Call) RunAndReturn(run func(context.Context, *proto.GetBlockRequest) (*proto.GetBlockResponse, error)) *NodeServer_GetBlock_Call {
	_c.Call.Return(run)
	return _c
}

// GetBlockInfo provides a mock function with given fields: _a0, _a1
func (_m *NodeServer) GetBlockInfo(_a0 context.Context, _a1 *proto.GetBlockInfoRequest) (*proto.GetBlockInfoResponse, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for GetBlockInfo")
	}

	var r0 *proto.GetBlockInfoResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *proto.GetBlockInfoRequest) (*proto.GetBlockInfoResponse, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *proto.GetBlockInfoRequest) *proto.GetBlockInfoResponse); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*proto.GetBlockInfoResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *proto.GetBlockInfoRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NodeServer_GetBlockInfo_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetBlockInfo'
type NodeServer_GetBlockInfo_Call struct {
	*mock.Call
}

// GetBlockInfo is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 *proto.GetBlockInfoRequest
func (_e *NodeServer_Expecter) GetBlockInfo(_a0 interface{}, _a1 interface{}) *NodeServer_GetBlockInfo_Call {
	return &NodeServer_GetBlockInfo_Call{Call: _e.mock.On("GetBlockInfo", _a0, _a1)}
}

func (_c *NodeServer_GetBlockInfo_Call) Run(run func(_a0 context.Context, _a1 *proto.GetBlockInfoRequest)) *NodeServer_GetBlockInfo_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*proto.GetBlockInfoRequest))
	})
	return _c
}

func (_c *NodeServer_GetBlockInfo_Call) Return(_a0 *proto.GetBlockInfoResponse, _a1 error) *NodeServer_GetBlockInfo_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *NodeServer_GetBlockInfo_Call) RunAndReturn(run func(context.Context, *proto.GetBlockInfoRequest) (*proto.GetBlockInfoResponse, error)) *NodeServer_GetBlockInfo_Call {
	_c.Call.Return(run)
	return _c
}

// GetBlockInfos provides a mock function with given fields: _a0, _a1
func (_m *NodeServer) GetBlockInfos(_a0 context.Context, _a1 *proto.GetBlockInfosRequest) (*proto.GetBlockInfosResponse, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for GetBlockInfos")
	}

	var r0 *proto.GetBlockInfosResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *proto.GetBlockInfosRequest) (*proto.GetBlockInfosResponse, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *proto.GetBlockInfosRequest) *proto.GetBlockInfosResponse); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*proto.GetBlockInfosResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *proto.GetBlockInfosRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NodeServer_GetBlockInfos_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetBlockInfos'
type NodeServer_GetBlockInfos_Call struct {
	*mock.Call
}

// GetBlockInfos is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 *proto.GetBlockInfosRequest
func (_e *NodeServer_Expecter) GetBlockInfos(_a0 interface{}, _a1 interface{}) *NodeServer_GetBlockInfos_Call {
	return &NodeServer_GetBlockInfos_Call{Call: _e.mock.On("GetBlockInfos", _a0, _a1)}
}

func (_c *NodeServer_GetBlockInfos_Call) Run(run func(_a0 context.Context, _a1 *proto.GetBlockInfosRequest)) *NodeServer_GetBlockInfos_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*proto.GetBlockInfosRequest))
	})
	return _c
}

func (_c *NodeServer_GetBlockInfos_Call) Return(_a0 *proto.GetBlockInfosResponse, _a1 error) *NodeServer_GetBlockInfos_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *NodeServer_GetBlockInfos_Call) RunAndReturn(run func(context.Context, *proto.GetBlockInfosRequest) (*proto.GetBlockInfosResponse, error)) *NodeServer_GetBlockInfos_Call {
	_c.Call.Return(run)
	return _c
}

// WriteBlock provides a mock function with given fields: _a0, _a1
func (_m *NodeServer) WriteBlock(_a0 context.Context, _a1 *proto.WriteBlockRequest) (*proto.WriteBlockResponse, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for WriteBlock")
	}

	var r0 *proto.WriteBlockResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *proto.WriteBlockRequest) (*proto.WriteBlockResponse, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *proto.WriteBlockRequest) *proto.WriteBlockResponse); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*proto.WriteBlockResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *proto.WriteBlockRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NodeServer_WriteBlock_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'WriteBlock'
type NodeServer_WriteBlock_Call struct {
	*mock.Call
}

// WriteBlock is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 *proto.WriteBlockRequest
func (_e *NodeServer_Expecter) WriteBlock(_a0 interface{}, _a1 interface{}) *NodeServer_WriteBlock_Call {
	return &NodeServer_WriteBlock_Call{Call: _e.mock.On("WriteBlock", _a0, _a1)}
}

func (_c *NodeServer_WriteBlock_Call) Run(run func(_a0 context.Context, _a1 *proto.WriteBlockRequest)) *NodeServer_WriteBlock_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*proto.WriteBlockRequest))
	})
	return _c
}

func (_c *NodeServer_WriteBlock_Call) Return(_a0 *proto.WriteBlockResponse, _a1 error) *NodeServer_WriteBlock_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *NodeServer_WriteBlock_Call) RunAndReturn(run func(context.Context, *proto.WriteBlockRequest) (*proto.WriteBlockResponse, error)) *NodeServer_WriteBlock_Call {
	_c.Call.Return(run)
	return _c
}

// mustEmbedUnimplementedNodeServer provides a mock function with no fields
func (_m *NodeServer) mustEmbedUnimplementedNodeServer() {
	_m.Called()
}

// NodeServer_mustEmbedUnimplementedNodeServer_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'mustEmbedUnimplementedNodeServer'
type NodeServer_mustEmbedUnimplementedNodeServer_Call struct {
	*mock.Call
}

// mustEmbedUnimplementedNodeServer is a helper method to define mock.On call
func (_e *NodeServer_Expecter) mustEmbedUnimplementedNodeServer() *NodeServer_mustEmbedUnimplementedNodeServer_Call {
	return &NodeServer_mustEmbedUnimplementedNodeServer_Call{Call: _e.mock.On("mustEmbedUnimplementedNodeServer")}
}

func (_c *NodeServer_mustEmbedUnimplementedNodeServer_Call) Run(run func()) *NodeServer_mustEmbedUnimplementedNodeServer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *NodeServer_mustEmbedUnimplementedNodeServer_Call) Return() *NodeServer_mustEmbedUnimplementedNodeServer_Call {
	_c.Call.Return()
	return _c
}

func (_c *NodeServer_mustEmbedUnimplementedNodeServer_Call) RunAndReturn(run func()) *NodeServer_mustEmbedUnimplementedNodeServer_Call {
	_c.Run(run)
	return _c
}

// NewNodeServer creates a new instance of NodeServer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewNodeServer(t interface {
	mock.TestingT
	Cleanup(func())
}) *NodeServer {
	mock := &NodeServer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
