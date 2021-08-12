package grpccontext

import (
	"context"

	"google.golang.org/grpc"
)

func ServerInterceptor(opts ...Option) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	o := defaultOptions()
	o.apply(opts...)
	return unaryServerInterceptor(o.contextFunc),
		streamServerInterceptor(o.contextFunc)
}

func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	o := defaultOptions()
	o.apply(opts...)
	return unaryServerInterceptor(o.contextFunc)
}

func StreamServerInterceptor(opts ...Option) grpc.StreamServerInterceptor {
	o := defaultOptions()
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
