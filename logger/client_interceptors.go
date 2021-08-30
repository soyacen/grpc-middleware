package grpclogger

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

func ClientInterceptor(loggerFactory LoggerFactory, opts ...Option) (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	o := defaultOptions()
	o.apply(opts...)
	return unaryClientInterceptor(loggerFactory, o), streamClientInterceptor(loggerFactory, o)
}

func UnaryClientInterceptor(loggerFactory LoggerFactory, opts ...Option) grpc.UnaryClientInterceptor {
	o := defaultOptions()
	o.apply(opts...)
	return unaryClientInterceptor(loggerFactory, o)
}

func StreamClientInterceptor(loggerFactory LoggerFactory, opts ...Option) grpc.StreamClientInterceptor {
	o := defaultOptions()
	o.apply(opts...)
	return streamClientInterceptor(loggerFactory, o)
}

func unaryClientInterceptor(loggerFactory LoggerFactory, o *options) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req interface{},
		reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if loggerFactory == nil {
			return invoker(ctx, method, req, reply, cc, opts...)
		}
		startTime := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		if o.skip(method, err) {
			return err
		}
		builder := NewFieldBuilder().
			Client().
			StartTime(startTime).
			Deadline(ctx).
			Method(method).
			PeerAddr(ctx).
			Status(err).
			Error(err).
			Latency(time.Since(startTime))
		logger := loggerFactory(ctx)
		logger.Log(builder.Build())
		return err
	}
}

func streamClientInterceptor(loggerFactory LoggerFactory, o *options) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		if loggerFactory == nil {
			return streamer(ctx, desc, cc, method, opts...)
		}
		startTime := time.Now()
		clientStream, err := streamer(ctx, desc, cc, method, opts...)
		if o.skip(method, err) {
			return clientStream, err
		}
		builder := NewFieldBuilder().
			Client().
			StartTime(startTime).
			Deadline(ctx).
			Method(method).
			PeerAddr(ctx).
			Status(err).
			Error(err).
			Latency(time.Since(startTime))
		logger := loggerFactory(ctx)
		logger.Log(builder.Build())
		return clientStream, err
	}
}
