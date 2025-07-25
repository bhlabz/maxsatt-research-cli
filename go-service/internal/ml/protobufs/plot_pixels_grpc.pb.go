// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.29.3
// source: plot_pixels.proto

package protobufs

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
	PlotPixelsService_PlotPixels_FullMethodName      = "/ml.PlotPixelsService/PlotPixels"
	PlotPixelsService_PlotDeltaPixels_FullMethodName = "/ml.PlotPixelsService/PlotDeltaPixels"
)

// PlotPixelsServiceClient is the client API for PlotPixelsService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type PlotPixelsServiceClient interface {
	PlotPixels(ctx context.Context, in *PlotPixelsRequest, opts ...grpc.CallOption) (*PlotPixelsResponse, error)
	PlotDeltaPixels(ctx context.Context, in *PlotDeltaPixelsRequest, opts ...grpc.CallOption) (*PlotDeltaPixelsResponse, error)
}

type plotPixelsServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewPlotPixelsServiceClient(cc grpc.ClientConnInterface) PlotPixelsServiceClient {
	return &plotPixelsServiceClient{cc}
}

func (c *plotPixelsServiceClient) PlotPixels(ctx context.Context, in *PlotPixelsRequest, opts ...grpc.CallOption) (*PlotPixelsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(PlotPixelsResponse)
	err := c.cc.Invoke(ctx, PlotPixelsService_PlotPixels_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *plotPixelsServiceClient) PlotDeltaPixels(ctx context.Context, in *PlotDeltaPixelsRequest, opts ...grpc.CallOption) (*PlotDeltaPixelsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(PlotDeltaPixelsResponse)
	err := c.cc.Invoke(ctx, PlotPixelsService_PlotDeltaPixels_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// PlotPixelsServiceServer is the server API for PlotPixelsService service.
// All implementations must embed UnimplementedPlotPixelsServiceServer
// for forward compatibility.
type PlotPixelsServiceServer interface {
	PlotPixels(context.Context, *PlotPixelsRequest) (*PlotPixelsResponse, error)
	PlotDeltaPixels(context.Context, *PlotDeltaPixelsRequest) (*PlotDeltaPixelsResponse, error)
	mustEmbedUnimplementedPlotPixelsServiceServer()
}

// UnimplementedPlotPixelsServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedPlotPixelsServiceServer struct{}

func (UnimplementedPlotPixelsServiceServer) PlotPixels(context.Context, *PlotPixelsRequest) (*PlotPixelsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PlotPixels not implemented")
}
func (UnimplementedPlotPixelsServiceServer) PlotDeltaPixels(context.Context, *PlotDeltaPixelsRequest) (*PlotDeltaPixelsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PlotDeltaPixels not implemented")
}
func (UnimplementedPlotPixelsServiceServer) mustEmbedUnimplementedPlotPixelsServiceServer() {}
func (UnimplementedPlotPixelsServiceServer) testEmbeddedByValue()                           {}

// UnsafePlotPixelsServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to PlotPixelsServiceServer will
// result in compilation errors.
type UnsafePlotPixelsServiceServer interface {
	mustEmbedUnimplementedPlotPixelsServiceServer()
}

func RegisterPlotPixelsServiceServer(s grpc.ServiceRegistrar, srv PlotPixelsServiceServer) {
	// If the following call pancis, it indicates UnimplementedPlotPixelsServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&PlotPixelsService_ServiceDesc, srv)
}

func _PlotPixelsService_PlotPixels_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PlotPixelsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PlotPixelsServiceServer).PlotPixels(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PlotPixelsService_PlotPixels_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PlotPixelsServiceServer).PlotPixels(ctx, req.(*PlotPixelsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PlotPixelsService_PlotDeltaPixels_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PlotDeltaPixelsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PlotPixelsServiceServer).PlotDeltaPixels(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PlotPixelsService_PlotDeltaPixels_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PlotPixelsServiceServer).PlotDeltaPixels(ctx, req.(*PlotDeltaPixelsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// PlotPixelsService_ServiceDesc is the grpc.ServiceDesc for PlotPixelsService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var PlotPixelsService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "ml.PlotPixelsService",
	HandlerType: (*PlotPixelsServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "PlotPixels",
			Handler:    _PlotPixelsService_PlotPixels_Handler,
		},
		{
			MethodName: "PlotDeltaPixels",
			Handler:    _PlotPixelsService_PlotDeltaPixels_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "plot_pixels.proto",
}
