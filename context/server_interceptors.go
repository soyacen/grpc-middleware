package grpccontext

import (
	"context"

	"google.golang.org/grpc"
)

// ServerInterceptor returns new unary and stream server interceptors for otel trace.
func ServerInterceptor(opts ...Option) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	o := defaultClientOptions()
	o.apply(opts...)
	return unaryServerInterceptor(o.contextFunc),
		streamServerInterceptor(o.contextFunc)
}

// UnaryServerInterceptor returns a new unary server interceptor for otel trace.
func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	o := defaultClientOptions()
	o.apply(opts...)
	return unaryServerInterceptor(o.contextFunc)
}

// StreamServerInterceptor returns a new streaming server interceptor for otel trace.
func StreamServerInterceptor(opts ...Option) grpc.StreamServerInterceptor {
	o := defaultClientOptions()
	o.apply(opts...)
	return streamServerInterceptor(o.contextFunc)
}

func unaryServerInterceptor(
	contextFunc ContextFunc,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		ctx = contextFunc(ctx)
		resp, err := handler(ctx, req)
		return resp, err
	}
}

func streamServerInterceptor(
	streamHandlerFunc ContextFunc,
) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := stream.Context()
		ctx = streamHandlerFunc(ctx)
		return handler(srv, stream)
	}
}
