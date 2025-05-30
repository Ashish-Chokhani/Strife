// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.29.3
// source: bank.proto

package protofiles

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	BankService_ProcessPayment_FullMethodName = "/proto.BankService/ProcessPayment"
	BankService_PrepareDebit_FullMethodName   = "/proto.BankService/PrepareDebit"
	BankService_PrepareCredit_FullMethodName  = "/proto.BankService/PrepareCredit"
	BankService_CommitDebit_FullMethodName    = "/proto.BankService/CommitDebit"
	BankService_CommitCredit_FullMethodName   = "/proto.BankService/CommitCredit"
	BankService_Abort_FullMethodName          = "/proto.BankService/Abort"
	BankService_GetBankName_FullMethodName    = "/proto.BankService/GetBankName"
)

// BankServiceClient is the client API for BankService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BankServiceClient interface {
	// Original method for simple payments
	ProcessPayment(ctx context.Context, in *PaymentRequest, opts ...grpc.CallOption) (*PaymentResponse, error)
	// New methods for two-phase commit
	PrepareDebit(ctx context.Context, in *TwoPhaseRequest, opts ...grpc.CallOption) (*TwoPhaseResponse, error)
	PrepareCredit(ctx context.Context, in *TwoPhaseRequest, opts ...grpc.CallOption) (*TwoPhaseResponse, error)
	CommitDebit(ctx context.Context, in *TwoPhaseRequest, opts ...grpc.CallOption) (*TwoPhaseResponse, error)
	CommitCredit(ctx context.Context, in *TwoPhaseRequest, opts ...grpc.CallOption) (*TwoPhaseResponse, error)
	Abort(ctx context.Context, in *TwoPhaseRequest, opts ...grpc.CallOption) (*TwoPhaseResponse, error)
	// Method to get bank name
	GetBankName(ctx context.Context, in *TwoPhaseRequest, opts ...grpc.CallOption) (*TwoPhaseResponse, error)
}

type bankServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewBankServiceClient(cc grpc.ClientConnInterface) BankServiceClient {
	return &bankServiceClient{cc}
}

func (c *bankServiceClient) ProcessPayment(ctx context.Context, in *PaymentRequest, opts ...grpc.CallOption) (*PaymentResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(PaymentResponse)
	err := c.cc.Invoke(ctx, BankService_ProcessPayment_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bankServiceClient) PrepareDebit(ctx context.Context, in *TwoPhaseRequest, opts ...grpc.CallOption) (*TwoPhaseResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(TwoPhaseResponse)
	err := c.cc.Invoke(ctx, BankService_PrepareDebit_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bankServiceClient) PrepareCredit(ctx context.Context, in *TwoPhaseRequest, opts ...grpc.CallOption) (*TwoPhaseResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(TwoPhaseResponse)
	err := c.cc.Invoke(ctx, BankService_PrepareCredit_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bankServiceClient) CommitDebit(ctx context.Context, in *TwoPhaseRequest, opts ...grpc.CallOption) (*TwoPhaseResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(TwoPhaseResponse)
	err := c.cc.Invoke(ctx, BankService_CommitDebit_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bankServiceClient) CommitCredit(ctx context.Context, in *TwoPhaseRequest, opts ...grpc.CallOption) (*TwoPhaseResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(TwoPhaseResponse)
	err := c.cc.Invoke(ctx, BankService_CommitCredit_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bankServiceClient) Abort(ctx context.Context, in *TwoPhaseRequest, opts ...grpc.CallOption) (*TwoPhaseResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(TwoPhaseResponse)
	err := c.cc.Invoke(ctx, BankService_Abort_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *bankServiceClient) GetBankName(ctx context.Context, in *TwoPhaseRequest, opts ...grpc.CallOption) (*TwoPhaseResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(TwoPhaseResponse)
	err := c.cc.Invoke(ctx, BankService_GetBankName_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BankServiceServer is the server API for BankService service.
// All implementations must embed UnimplementedBankServiceServer
// for forward compatibility.
type BankServiceServer interface {
	// Original method for simple payments
	ProcessPayment(context.Context, *PaymentRequest) (*PaymentResponse, error)
	// New methods for two-phase commit
	PrepareDebit(context.Context, *TwoPhaseRequest) (*TwoPhaseResponse, error)
	PrepareCredit(context.Context, *TwoPhaseRequest) (*TwoPhaseResponse, error)
	CommitDebit(context.Context, *TwoPhaseRequest) (*TwoPhaseResponse, error)
	CommitCredit(context.Context, *TwoPhaseRequest) (*TwoPhaseResponse, error)
	Abort(context.Context, *TwoPhaseRequest) (*TwoPhaseResponse, error)
	// Method to get bank name
	GetBankName(context.Context, *TwoPhaseRequest) (*TwoPhaseResponse, error)
	mustEmbedUnimplementedBankServiceServer()
}

// UnimplementedBankServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedBankServiceServer struct{}

func (UnimplementedBankServiceServer) ProcessPayment(context.Context, *PaymentRequest) (*PaymentResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ProcessPayment not implemented")
}
func (UnimplementedBankServiceServer) PrepareDebit(context.Context, *TwoPhaseRequest) (*TwoPhaseResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PrepareDebit not implemented")
}
func (UnimplementedBankServiceServer) PrepareCredit(context.Context, *TwoPhaseRequest) (*TwoPhaseResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PrepareCredit not implemented")
}
func (UnimplementedBankServiceServer) CommitDebit(context.Context, *TwoPhaseRequest) (*TwoPhaseResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CommitDebit not implemented")
}
func (UnimplementedBankServiceServer) CommitCredit(context.Context, *TwoPhaseRequest) (*TwoPhaseResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CommitCredit not implemented")
}
func (UnimplementedBankServiceServer) Abort(context.Context, *TwoPhaseRequest) (*TwoPhaseResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Abort not implemented")
}
func (UnimplementedBankServiceServer) GetBankName(context.Context, *TwoPhaseRequest) (*TwoPhaseResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBankName not implemented")
}
func (UnimplementedBankServiceServer) mustEmbedUnimplementedBankServiceServer() {}
func (UnimplementedBankServiceServer) testEmbeddedByValue()                     {}

// UnsafeBankServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BankServiceServer will
// result in compilation errors.
type UnsafeBankServiceServer interface {
	mustEmbedUnimplementedBankServiceServer()
}

func RegisterBankServiceServer(s grpc.ServiceRegistrar, srv BankServiceServer) {
	// If the following call pancis, it indicates UnimplementedBankServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&BankService_ServiceDesc, srv)
}

func _BankService_ProcessPayment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PaymentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BankServiceServer).ProcessPayment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BankService_ProcessPayment_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BankServiceServer).ProcessPayment(ctx, req.(*PaymentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BankService_PrepareDebit_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TwoPhaseRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BankServiceServer).PrepareDebit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BankService_PrepareDebit_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BankServiceServer).PrepareDebit(ctx, req.(*TwoPhaseRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BankService_PrepareCredit_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TwoPhaseRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BankServiceServer).PrepareCredit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BankService_PrepareCredit_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BankServiceServer).PrepareCredit(ctx, req.(*TwoPhaseRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BankService_CommitDebit_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TwoPhaseRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BankServiceServer).CommitDebit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BankService_CommitDebit_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BankServiceServer).CommitDebit(ctx, req.(*TwoPhaseRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BankService_CommitCredit_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TwoPhaseRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BankServiceServer).CommitCredit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BankService_CommitCredit_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BankServiceServer).CommitCredit(ctx, req.(*TwoPhaseRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BankService_Abort_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TwoPhaseRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BankServiceServer).Abort(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BankService_Abort_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BankServiceServer).Abort(ctx, req.(*TwoPhaseRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BankService_GetBankName_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TwoPhaseRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BankServiceServer).GetBankName(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: BankService_GetBankName_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BankServiceServer).GetBankName(ctx, req.(*TwoPhaseRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// BankService_ServiceDesc is the grpc.ServiceDesc for BankService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var BankService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proto.BankService",
	HandlerType: (*BankServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ProcessPayment",
			Handler:    _BankService_ProcessPayment_Handler,
		},
		{
			MethodName: "PrepareDebit",
			Handler:    _BankService_PrepareDebit_Handler,
		},
		{
			MethodName: "PrepareCredit",
			Handler:    _BankService_PrepareCredit_Handler,
		},
		{
			MethodName: "CommitDebit",
			Handler:    _BankService_CommitDebit_Handler,
		},
		{
			MethodName: "CommitCredit",
			Handler:    _BankService_CommitCredit_Handler,
		},
		{
			MethodName: "Abort",
			Handler:    _BankService_Abort_Handler,
		},
		{
			MethodName: "GetBankName",
			Handler:    _BankService_GetBankName_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "bank.proto",
}
