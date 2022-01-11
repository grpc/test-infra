// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.12.4
// source: endpoint.proto

package endpointupdater

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// EndpointUpdaterClient is the client API for EndpointUpdater service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type EndpointUpdaterClient interface {
	// Sends an update
	UpdateEndpoint(ctx context.Context, in *EndpointUpdaterRequest, opts ...grpc.CallOption) (*EndpointUpdaterReply, error)
	QuitEndpointUpdateServer(ctx context.Context, in *Void, opts ...grpc.CallOption) (*Void, error)
}

type endpointUpdaterClient struct {
	cc grpc.ClientConnInterface
}

func NewEndpointUpdaterClient(cc grpc.ClientConnInterface) EndpointUpdaterClient {
	return &endpointUpdaterClient{cc}
}

func (c *endpointUpdaterClient) UpdateEndpoint(ctx context.Context, in *EndpointUpdaterRequest, opts ...grpc.CallOption) (*EndpointUpdaterReply, error) {
	out := new(EndpointUpdaterReply)
	err := c.cc.Invoke(ctx, "/endpointupdater.EndpointUpdater/UpdateEndpoint", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *endpointUpdaterClient) QuitEndpointUpdateServer(ctx context.Context, in *Void, opts ...grpc.CallOption) (*Void, error) {
	out := new(Void)
	err := c.cc.Invoke(ctx, "/endpointupdater.EndpointUpdater/QuitEndpointUpdateServer", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// EndpointUpdaterServer is the server API for EndpointUpdater service.
// All implementations must embed UnimplementedEndpointUpdaterServer
// for forward compatibility
type EndpointUpdaterServer interface {
	// Sends an update
	UpdateEndpoint(context.Context, *EndpointUpdaterRequest) (*EndpointUpdaterReply, error)
	QuitEndpointUpdateServer(context.Context, *Void) (*Void, error)
	mustEmbedUnimplementedEndpointUpdaterServer()
}

// UnimplementedEndpointUpdaterServer must be embedded to have forward compatible implementations.
type UnimplementedEndpointUpdaterServer struct {
}

func (UnimplementedEndpointUpdaterServer) UpdateEndpoint(context.Context, *EndpointUpdaterRequest) (*EndpointUpdaterReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateEndpoint not implemented")
}
func (UnimplementedEndpointUpdaterServer) QuitEndpointUpdateServer(context.Context, *Void) (*Void, error) {
	return nil, status.Errorf(codes.Unimplemented, "method QuitEndpointUpdateServer not implemented")
}
func (UnimplementedEndpointUpdaterServer) mustEmbedUnimplementedEndpointUpdaterServer() {}

// UnsafeEndpointUpdaterServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to EndpointUpdaterServer will
// result in compilation errors.
type UnsafeEndpointUpdaterServer interface {
	mustEmbedUnimplementedEndpointUpdaterServer()
}

func RegisterEndpointUpdaterServer(s grpc.ServiceRegistrar, srv EndpointUpdaterServer) {
	s.RegisterService(&EndpointUpdater_ServiceDesc, srv)
}

func _EndpointUpdater_UpdateEndpoint_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EndpointUpdaterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EndpointUpdaterServer).UpdateEndpoint(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/endpointupdater.EndpointUpdater/UpdateEndpoint",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EndpointUpdaterServer).UpdateEndpoint(ctx, req.(*EndpointUpdaterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EndpointUpdater_QuitEndpointUpdateServer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Void)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EndpointUpdaterServer).QuitEndpointUpdateServer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/endpointupdater.EndpointUpdater/QuitEndpointUpdateServer",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EndpointUpdaterServer).QuitEndpointUpdateServer(ctx, req.(*Void))
	}
	return interceptor(ctx, in, info, handler)
}

// EndpointUpdater_ServiceDesc is the grpc.ServiceDesc for EndpointUpdater service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var EndpointUpdater_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "endpointupdater.EndpointUpdater",
	HandlerType: (*EndpointUpdaterServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "UpdateEndpoint",
			Handler:    _EndpointUpdater_UpdateEndpoint_Handler,
		},
		{
			MethodName: "QuitEndpointUpdateServer",
			Handler:    _EndpointUpdater_QuitEndpointUpdateServer_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "endpoint.proto",
}
