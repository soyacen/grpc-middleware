package grpclogger

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

func ClientInterceptor(opts ...Option) (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	o := defaultOptions()
	o.apply(opts...)
	return unaryClientInterceptor(o), streamClientInterceptor(o)
}

func UnaryClientInterceptor(opts ...Option) grpc.UnaryClientInterceptor {
	o := defaultOptions()
	o.apply(opts...)
	return unaryClientInterceptor(o)
}

func StreamClientInterceptor(opts ...Option) grpc.StreamClientInterceptor {
	o := defaultOptions()
	o.apply(opts...)
	return streamClientInterceptor(o)
}

func unaryClientInterceptor(o *options) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req interface{},
		reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if o.loggerFactory == nil {
			return invoker(ctx, method, req, reply, cc, opts...)
		}
		logger := o.loggerFactory(ctx)
		startTime := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		builder := NewFieldBuilder().
			Client().
			StartTime(startTime).
			Deadline(ctx).
			Method(method).
			PeerAddr(ctx).
			Status(err).
			Error(err).
			Latency(time.Since(startTime))
		logger.Log(builder.Build())
		return err
	}
}

func streamClientInterceptor(o *options) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		if o.loggerFactory == nil {
			return streamer(ctx, desc, cc, method, opts...)
		}
		logger := o.loggerFactory(ctx)
		startTime := time.Now()
		clientStream, err := streamer(ctx, desc, cc, method, opts...)
		builder := NewFieldBuilder().
			Client().
			StartTime(startTime).
			Deadline(ctx).
			Method(method).
			PeerAddr(ctx).
			Status(err).
			Error(err).
			Latency(time.Since(startTime))
		logger.Log(builder.Build())
		return clientStream, err
	}
}
