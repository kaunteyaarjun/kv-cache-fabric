// Code generated for kvblock gRPC service. DO NOT EDIT.
package kvblock

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// BlockManagerServiceClient is the client API for BlockManagerService service.
type BlockManagerServiceClient interface {
	AllocateBlock(ctx context.Context, in *AllocateBlockRequest, opts ...grpc.CallOption) (*AllocateBlockResponse, error)
	TouchBlock(ctx context.Context, in *TouchBlockRequest, opts ...grpc.CallOption) (*TouchBlockResponse, error)
	GetBlockLocation(ctx context.Context, in *GetBlockLocationRequest, opts ...grpc.CallOption) (*GetBlockLocationResponse, error)
	EvictLRU(ctx context.Context, in *EvictLRURequest, opts ...grpc.CallOption) (*EvictLRUResponse, error)
	WriteBlockData(ctx context.Context, in *WriteBlockDataRequest, opts ...grpc.CallOption) (*WriteBlockDataResponse, error)
	ReadBlockData(ctx context.Context, in *ReadBlockDataRequest, opts ...grpc.CallOption) (*ReadBlockDataResponse, error)
}

type blockManagerServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewBlockManagerServiceClient(cc grpc.ClientConnInterface) BlockManagerServiceClient {
	return &blockManagerServiceClient{cc}
}

func (c *blockManagerServiceClient) AllocateBlock(ctx context.Context, in *AllocateBlockRequest, opts ...grpc.CallOption) (*AllocateBlockResponse, error) {
	out := new(AllocateBlockResponse)
	err := c.cc.Invoke(ctx, "/kvblock.BlockManagerService/AllocateBlock", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blockManagerServiceClient) TouchBlock(ctx context.Context, in *TouchBlockRequest, opts ...grpc.CallOption) (*TouchBlockResponse, error) {
	out := new(TouchBlockResponse)
	err := c.cc.Invoke(ctx, "/kvblock.BlockManagerService/TouchBlock", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blockManagerServiceClient) GetBlockLocation(ctx context.Context, in *GetBlockLocationRequest, opts ...grpc.CallOption) (*GetBlockLocationResponse, error) {
	out := new(GetBlockLocationResponse)
	err := c.cc.Invoke(ctx, "/kvblock.BlockManagerService/GetBlockLocation", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blockManagerServiceClient) EvictLRU(ctx context.Context, in *EvictLRURequest, opts ...grpc.CallOption) (*EvictLRUResponse, error) {
	out := new(EvictLRUResponse)
	err := c.cc.Invoke(ctx, "/kvblock.BlockManagerService/EvictLRU", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blockManagerServiceClient) WriteBlockData(ctx context.Context, in *WriteBlockDataRequest, opts ...grpc.CallOption) (*WriteBlockDataResponse, error) {
	out := new(WriteBlockDataResponse)
	err := c.cc.Invoke(ctx, "/kvblock.BlockManagerService/WriteBlockData", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blockManagerServiceClient) ReadBlockData(ctx context.Context, in *ReadBlockDataRequest, opts ...grpc.CallOption) (*ReadBlockDataResponse, error) {
	out := new(ReadBlockDataResponse)
	err := c.cc.Invoke(ctx, "/kvblock.BlockManagerService/ReadBlockData", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BlockManagerServiceServer is the server API for BlockManagerService service.
type BlockManagerServiceServer interface {
	AllocateBlock(context.Context, *AllocateBlockRequest) (*AllocateBlockResponse, error)
	TouchBlock(context.Context, *TouchBlockRequest) (*TouchBlockResponse, error)
	GetBlockLocation(context.Context, *GetBlockLocationRequest) (*GetBlockLocationResponse, error)
	EvictLRU(context.Context, *EvictLRURequest) (*EvictLRUResponse, error)
	WriteBlockData(context.Context, *WriteBlockDataRequest) (*WriteBlockDataResponse, error)
	ReadBlockData(context.Context, *ReadBlockDataRequest) (*ReadBlockDataResponse, error)
	mustEmbedUnimplementedBlockManagerServiceServer()
}

// UnimplementedBlockManagerServiceServer must be embedded to have forward compatible implementations.
type UnimplementedBlockManagerServiceServer struct{}

func (UnimplementedBlockManagerServiceServer) AllocateBlock(context.Context, *AllocateBlockRequest) (*AllocateBlockResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AllocateBlock not implemented")
}
func (UnimplementedBlockManagerServiceServer) TouchBlock(context.Context, *TouchBlockRequest) (*TouchBlockResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TouchBlock not implemented")
}
func (UnimplementedBlockManagerServiceServer) GetBlockLocation(context.Context, *GetBlockLocationRequest) (*GetBlockLocationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBlockLocation not implemented")
}
func (UnimplementedBlockManagerServiceServer) EvictLRU(context.Context, *EvictLRURequest) (*EvictLRUResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method EvictLRU not implemented")
}
func (UnimplementedBlockManagerServiceServer) WriteBlockData(context.Context, *WriteBlockDataRequest) (*WriteBlockDataResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method WriteBlockData not implemented")
}
func (UnimplementedBlockManagerServiceServer) ReadBlockData(context.Context, *ReadBlockDataRequest) (*ReadBlockDataResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReadBlockData not implemented")
}
func (UnimplementedBlockManagerServiceServer) mustEmbedUnimplementedBlockManagerServiceServer() {}

// UnsafeBlockManagerServiceServer may be embedded to opt out of forward compatibility for this service.
type UnsafeBlockManagerServiceServer interface {
	mustEmbedUnimplementedBlockManagerServiceServer()
}

func RegisterBlockManagerServiceServer(s grpc.ServiceRegistrar, srv BlockManagerServiceServer) {
	s.RegisterService(&BlockManagerService_ServiceDesc, srv)
}

func _BlockManagerService_AllocateBlock_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AllocateBlockRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlockManagerServiceServer).AllocateBlock(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/kvblock.BlockManagerService/AllocateBlock",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlockManagerServiceServer).AllocateBlock(ctx, req.(*AllocateBlockRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlockManagerService_TouchBlock_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TouchBlockRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlockManagerServiceServer).TouchBlock(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/kvblock.BlockManagerService/TouchBlock",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlockManagerServiceServer).TouchBlock(ctx, req.(*TouchBlockRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlockManagerService_GetBlockLocation_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetBlockLocationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlockManagerServiceServer).GetBlockLocation(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/kvblock.BlockManagerService/GetBlockLocation",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlockManagerServiceServer).GetBlockLocation(ctx, req.(*GetBlockLocationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlockManagerService_EvictLRU_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EvictLRURequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlockManagerServiceServer).EvictLRU(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/kvblock.BlockManagerService/EvictLRU",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlockManagerServiceServer).EvictLRU(ctx, req.(*EvictLRURequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlockManagerService_WriteBlockData_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(WriteBlockDataRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlockManagerServiceServer).WriteBlockData(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/kvblock.BlockManagerService/WriteBlockData",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlockManagerServiceServer).WriteBlockData(ctx, req.(*WriteBlockDataRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlockManagerService_ReadBlockData_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReadBlockDataRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlockManagerServiceServer).ReadBlockData(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/kvblock.BlockManagerService/ReadBlockData",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlockManagerServiceServer).ReadBlockData(ctx, req.(*ReadBlockDataRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var BlockManagerService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "kvblock.BlockManagerService",
	HandlerType: (*BlockManagerServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AllocateBlock",
			Handler:    _BlockManagerService_AllocateBlock_Handler,
		},
		{
			MethodName: "TouchBlock",
			Handler:    _BlockManagerService_TouchBlock_Handler,
		},
		{
			MethodName: "GetBlockLocation",
			Handler:    _BlockManagerService_GetBlockLocation_Handler,
		},
		{
			MethodName: "EvictLRU",
			Handler:    _BlockManagerService_EvictLRU_Handler,
		},
		{
			MethodName: "WriteBlockData",
			Handler:    _BlockManagerService_WriteBlockData_Handler,
		},
		{
			MethodName: "ReadBlockData",
			Handler:    _BlockManagerService_ReadBlockData_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/kvblock/kvblock.proto",
}
