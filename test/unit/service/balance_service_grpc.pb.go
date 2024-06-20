// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v5.27.1
// source: balance_service.proto

package service

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// BalanceServiceClient is the client API for BalanceService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BalanceServiceClient interface {
	Balance(ctx context.Context, in *BalanceRequest, opts ...grpc.CallOption) (*BalanceResponse, error)
	AllBalances(ctx context.Context, in *AllBalancesRequest, opts ...grpc.CallOption) (*AllBalancesResponse, error)
	AllowedBalance(ctx context.Context, in *AllowedBalanceRequest, opts ...grpc.CallOption) (*AllowedBalanceResponse, error)
	AddBalanceByAdmin(ctx context.Context, in *BalanceAdjustmentRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	SubtractBalanceByAdmin(ctx context.Context, in *BalanceAdjustmentRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	AddAllowedBalanceByAdmin(ctx context.Context, in *AllowedBalanceAdjustmentRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	SubtractAllowedBalanceByAdmin(ctx context.Context, in *AllowedBalanceAdjustmentRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
	UnlockAllowedBalanceByAdmin(ctx context.Context, in *AllowedBalanceUnlockRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type balanceServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewBalanceServiceClient(cc grpc.ClientConnInterface) BalanceServiceClient {
	return &balanceServiceClient{cc}
}

func (c *balanceServiceClient) Balance(ctx context.Context, in *BalanceRequest, opts ...grpc.CallOption) (*BalanceResponse, error) {
	out := new(BalanceResponse)
	err := c.cc.Invoke(ctx, "/foundationtoken.BalanceService/Balance", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *balanceServiceClient) AllBalances(ctx context.Context, in *AllBalancesRequest, opts ...grpc.CallOption) (*AllBalancesResponse, error) {
	out := new(AllBalancesResponse)
	err := c.cc.Invoke(ctx, "/foundationtoken.BalanceService/AllBalances", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *balanceServiceClient) AllowedBalance(ctx context.Context, in *AllowedBalanceRequest, opts ...grpc.CallOption) (*AllowedBalanceResponse, error) {
	out := new(AllowedBalanceResponse)
	err := c.cc.Invoke(ctx, "/foundationtoken.BalanceService/AllowedBalance", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *balanceServiceClient) AddBalanceByAdmin(ctx context.Context, in *BalanceAdjustmentRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/foundationtoken.BalanceService/AddBalanceByAdmin", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *balanceServiceClient) SubtractBalanceByAdmin(ctx context.Context, in *BalanceAdjustmentRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/foundationtoken.BalanceService/SubtractBalanceByAdmin", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *balanceServiceClient) AddAllowedBalanceByAdmin(ctx context.Context, in *AllowedBalanceAdjustmentRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/foundationtoken.BalanceService/AddAllowedBalanceByAdmin", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *balanceServiceClient) SubtractAllowedBalanceByAdmin(ctx context.Context, in *AllowedBalanceAdjustmentRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/foundationtoken.BalanceService/SubtractAllowedBalanceByAdmin", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *balanceServiceClient) UnlockAllowedBalanceByAdmin(ctx context.Context, in *AllowedBalanceUnlockRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/foundationtoken.BalanceService/UnlockAllowedBalanceByAdmin", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BalanceServiceServer is the server API for BalanceService service.
// All implementations must embed UnimplementedBalanceServiceServer
// for forward compatibility
type BalanceServiceServer interface {
	Balance(context.Context, *BalanceRequest) (*BalanceResponse, error)
	AllBalances(context.Context, *AllBalancesRequest) (*AllBalancesResponse, error)
	AllowedBalance(context.Context, *AllowedBalanceRequest) (*AllowedBalanceResponse, error)
	AddBalanceByAdmin(context.Context, *BalanceAdjustmentRequest) (*emptypb.Empty, error)
	SubtractBalanceByAdmin(context.Context, *BalanceAdjustmentRequest) (*emptypb.Empty, error)
	AddAllowedBalanceByAdmin(context.Context, *AllowedBalanceAdjustmentRequest) (*emptypb.Empty, error)
	SubtractAllowedBalanceByAdmin(context.Context, *AllowedBalanceAdjustmentRequest) (*emptypb.Empty, error)
	UnlockAllowedBalanceByAdmin(context.Context, *AllowedBalanceUnlockRequest) (*emptypb.Empty, error)
	mustEmbedUnimplementedBalanceServiceServer()
}

// UnimplementedBalanceServiceServer must be embedded to have forward compatible implementations.
type UnimplementedBalanceServiceServer struct {
}

func (UnimplementedBalanceServiceServer) Balance(context.Context, *BalanceRequest) (*BalanceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Balance not implemented")
}
func (UnimplementedBalanceServiceServer) AllBalances(context.Context, *AllBalancesRequest) (*AllBalancesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AllBalances not implemented")
}
func (UnimplementedBalanceServiceServer) AllowedBalance(context.Context, *AllowedBalanceRequest) (*AllowedBalanceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AllowedBalance not implemented")
}
func (UnimplementedBalanceServiceServer) AddBalanceByAdmin(context.Context, *BalanceAdjustmentRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddBalanceByAdmin not implemented")
}
func (UnimplementedBalanceServiceServer) SubtractBalanceByAdmin(context.Context, *BalanceAdjustmentRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SubtractBalanceByAdmin not implemented")
}
func (UnimplementedBalanceServiceServer) AddAllowedBalanceByAdmin(context.Context, *AllowedBalanceAdjustmentRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddAllowedBalanceByAdmin not implemented")
}
func (UnimplementedBalanceServiceServer) SubtractAllowedBalanceByAdmin(context.Context, *AllowedBalanceAdjustmentRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SubtractAllowedBalanceByAdmin not implemented")
}
func (UnimplementedBalanceServiceServer) UnlockAllowedBalanceByAdmin(context.Context, *AllowedBalanceUnlockRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UnlockAllowedBalanceByAdmin not implemented")
}
func (UnimplementedBalanceServiceServer) mustEmbedUnimplementedBalanceServiceServer() {}

// UnsafeBalanceServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BalanceServiceServer will
// result in compilation errors.
type UnsafeBalanceServiceServer interface {
	mustEmbedUnimplementedBalanceServiceServer()
}

func RegisterBalanceServiceServer(s grpc.ServiceRegistrar, srv BalanceServiceServer) {
	s.RegisterService(&BalanceService_ServiceDesc, srv)
}

func _BalanceService_Balance_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BalanceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BalanceServiceServer).Balance(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/foundationtoken.BalanceService/Balance",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BalanceServiceServer).Balance(ctx, req.(*BalanceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BalanceService_AllBalances_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AllBalancesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BalanceServiceServer).AllBalances(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/foundationtoken.BalanceService/AllBalances",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BalanceServiceServer).AllBalances(ctx, req.(*AllBalancesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BalanceService_AllowedBalance_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AllowedBalanceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BalanceServiceServer).AllowedBalance(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/foundationtoken.BalanceService/AllowedBalance",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BalanceServiceServer).AllowedBalance(ctx, req.(*AllowedBalanceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BalanceService_AddBalanceByAdmin_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BalanceAdjustmentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BalanceServiceServer).AddBalanceByAdmin(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/foundationtoken.BalanceService/AddBalanceByAdmin",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BalanceServiceServer).AddBalanceByAdmin(ctx, req.(*BalanceAdjustmentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BalanceService_SubtractBalanceByAdmin_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BalanceAdjustmentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BalanceServiceServer).SubtractBalanceByAdmin(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/foundationtoken.BalanceService/SubtractBalanceByAdmin",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BalanceServiceServer).SubtractBalanceByAdmin(ctx, req.(*BalanceAdjustmentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BalanceService_AddAllowedBalanceByAdmin_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AllowedBalanceAdjustmentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BalanceServiceServer).AddAllowedBalanceByAdmin(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/foundationtoken.BalanceService/AddAllowedBalanceByAdmin",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BalanceServiceServer).AddAllowedBalanceByAdmin(ctx, req.(*AllowedBalanceAdjustmentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BalanceService_SubtractAllowedBalanceByAdmin_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AllowedBalanceAdjustmentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BalanceServiceServer).SubtractAllowedBalanceByAdmin(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/foundationtoken.BalanceService/SubtractAllowedBalanceByAdmin",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BalanceServiceServer).SubtractAllowedBalanceByAdmin(ctx, req.(*AllowedBalanceAdjustmentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BalanceService_UnlockAllowedBalanceByAdmin_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AllowedBalanceUnlockRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BalanceServiceServer).UnlockAllowedBalanceByAdmin(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/foundationtoken.BalanceService/UnlockAllowedBalanceByAdmin",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BalanceServiceServer).UnlockAllowedBalanceByAdmin(ctx, req.(*AllowedBalanceUnlockRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// BalanceService_ServiceDesc is the grpc.ServiceDesc for BalanceService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var BalanceService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "foundationtoken.BalanceService",
	HandlerType: (*BalanceServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Balance",
			Handler:    _BalanceService_Balance_Handler,
		},
		{
			MethodName: "AllBalances",
			Handler:    _BalanceService_AllBalances_Handler,
		},
		{
			MethodName: "AllowedBalance",
			Handler:    _BalanceService_AllowedBalance_Handler,
		},
		{
			MethodName: "AddBalanceByAdmin",
			Handler:    _BalanceService_AddBalanceByAdmin_Handler,
		},
		{
			MethodName: "SubtractBalanceByAdmin",
			Handler:    _BalanceService_SubtractBalanceByAdmin_Handler,
		},
		{
			MethodName: "AddAllowedBalanceByAdmin",
			Handler:    _BalanceService_AddAllowedBalanceByAdmin_Handler,
		},
		{
			MethodName: "SubtractAllowedBalanceByAdmin",
			Handler:    _BalanceService_SubtractAllowedBalanceByAdmin_Handler,
		},
		{
			MethodName: "UnlockAllowedBalanceByAdmin",
			Handler:    _BalanceService_UnlockAllowedBalanceByAdmin_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "balance_service.proto",
}
