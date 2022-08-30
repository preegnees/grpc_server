// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.5
// source: cmd/proto/service.proto

package __

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

// MyServiceClient is the client API for MyService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type MyServiceClient interface {
	Streaming(ctx context.Context, opts ...grpc.CallOption) (MyService_StreamingClient, error)
}

type myServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewMyServiceClient(cc grpc.ClientConnInterface) MyServiceClient {
	return &myServiceClient{cc}
}

func (c *myServiceClient) Streaming(ctx context.Context, opts ...grpc.CallOption) (MyService_StreamingClient, error) {
	stream, err := c.cc.NewStream(ctx, &MyService_ServiceDesc.Streams[0], "/service.MyService/Streaming", opts...)
	if err != nil {
		return nil, err
	}
	x := &myServiceStreamingClient{stream}
	return x, nil
}

type MyService_StreamingClient interface {
	Send(*Req) error
	Recv() (*Res, error)
	grpc.ClientStream
}

type myServiceStreamingClient struct {
	grpc.ClientStream
}

func (x *myServiceStreamingClient) Send(m *Req) error {
	return x.ClientStream.SendMsg(m)
}

func (x *myServiceStreamingClient) Recv() (*Res, error) {
	m := new(Res)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// MyServiceServer is the server API for MyService service.
// All implementations must embed UnimplementedMyServiceServer
// for forward compatibility
type MyServiceServer interface {
	Streaming(MyService_StreamingServer) error
	mustEmbedUnimplementedMyServiceServer()
}

// UnimplementedMyServiceServer must be embedded to have forward compatible implementations.
type UnimplementedMyServiceServer struct {
}

func (UnimplementedMyServiceServer) Streaming(MyService_StreamingServer) error {
	return status.Errorf(codes.Unimplemented, "method Streaming not implemented")
}
func (UnimplementedMyServiceServer) mustEmbedUnimplementedMyServiceServer() {}

// UnsafeMyServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to MyServiceServer will
// result in compilation errors.
type UnsafeMyServiceServer interface {
	mustEmbedUnimplementedMyServiceServer()
}

func RegisterMyServiceServer(s grpc.ServiceRegistrar, srv MyServiceServer) {
	s.RegisterService(&MyService_ServiceDesc, srv)
}

func _MyService_Streaming_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(MyServiceServer).Streaming(&myServiceStreamingServer{stream})
}

type MyService_StreamingServer interface {
	Send(*Res) error
	Recv() (*Req, error)
	grpc.ServerStream
}

type myServiceStreamingServer struct {
	grpc.ServerStream
}

func (x *myServiceStreamingServer) Send(m *Res) error {
	return x.ServerStream.SendMsg(m)
}

func (x *myServiceStreamingServer) Recv() (*Req, error) {
	m := new(Req)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// MyService_ServiceDesc is the grpc.ServiceDesc for MyService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var MyService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "service.MyService",
	HandlerType: (*MyServiceServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Streaming",
			Handler:       _MyService_Streaming_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "cmd/proto/service.proto",
}
