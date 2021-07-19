package grpclogger

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

func ServerInterceptor(opts ...Option) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	o := defaultOptions()
	o.apply(opts...)
	return unaryServerInterceptor(o), streamServerInterceptor(o)
}

func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	o := defaultOptions()
	o.apply(opts...)
	return unaryServerInterceptor(o)
}

func StreamServerInterceptor(opts ...Option) grpc.StreamServerInterceptor {
	o := defaultOptions()
	o.apply(opts...)
	return streamServerInterceptor(o)
}

func unaryServerInterceptor(o *options) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if o.loggerFactory == nil {
			return handler(ctx, req)
		}
		logger := o.loggerFactory(ctx)
		startTime := time.Now()
		resp, err := handler(ctx, req)
		builder := NewFieldBuilder()
		builder.
			Server().
			StartTime(startTime).
			Deadline(ctx).
			Method(info.FullMethod).
			PeerAddr(ctx).
			Status(err).
			Error(err).
			Latency(time.Since(startTime))
		logger.Log(builder.Build())
		return resp, err
	}
}

func streamServerInterceptor(o *options) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if o.loggerFactory == nil {
			return handler(srv, stream)
		}
		logger := o.loggerFactory(stream.Context())
		startTime := time.Now()
		err := handler(srv, stream)
		builder := NewFieldBuilder()
		builder.
			Server().
			StartTime(startTime).
			Deadline(stream.Context()).
			Method(info.FullMethod).
			PeerAddr(stream.Context()).
			Status(err).
			Error(err).
			Latency(time.Since(startTime))
		logger.Log(builder.Build())
		return err
	}
}
