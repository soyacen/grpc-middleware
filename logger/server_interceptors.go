package grpclogger

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

func ServerInterceptor(loggerFactory LoggerFactory, opts ...Option) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	o := defaultOptions()
	o.apply(opts...)
	return unaryServerInterceptor(loggerFactory, o), streamServerInterceptor(loggerFactory, o)
}

func UnaryServerInterceptor(loggerFactory LoggerFactory, opts ...Option) grpc.UnaryServerInterceptor {
	o := defaultOptions()
	o.apply(opts...)
	return unaryServerInterceptor(loggerFactory, o)
}

func StreamServerInterceptor(loggerFactory LoggerFactory, opts ...Option) grpc.StreamServerInterceptor {
	o := defaultOptions()
	o.apply(opts...)
	return streamServerInterceptor(loggerFactory, o)
}

func unaryServerInterceptor(loggerFactory LoggerFactory, o *options) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if loggerFactory == nil {
			return handler(ctx, req)
		}
		startTime := time.Now()
		resp, err := handler(ctx, req)
		if o.skip(info.FullMethod, err) {
			return resp, err
		}
		builder := NewFieldBuilder().
			Server().
			StartTime(startTime).
			Deadline(ctx).
			Method(info.FullMethod).
			PeerAddr(ctx).
			Status(err).
			Error(err).
			Latency(time.Since(startTime))
		logger := loggerFactory(ctx)
		logger.Log(builder.Build())
		return resp, err
	}
}

func streamServerInterceptor(loggerFactory LoggerFactory, o *options) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if loggerFactory == nil {
			return handler(srv, stream)
		}
		startTime := time.Now()
		err := handler(srv, stream)
		if o.skip(info.FullMethod, err) {
			return err
		}
		builder := NewFieldBuilder().
			Server().
			StartTime(startTime).
			Deadline(stream.Context()).
			Method(info.FullMethod).
			PeerAddr(stream.Context()).
			Status(err).
			Error(err).
			Latency(time.Since(startTime))
		logger := loggerFactory(stream.Context())
		logger.Log(builder.Build())
		return err
	}
}
