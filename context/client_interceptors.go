package grpccontext

import (
	"context"

	"google.golang.org/grpc"
)

func ClientInterceptor(opts ...Option) (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	o := defaultClientOptions()
	o.apply(opts...)
	return unaryClientInterceptor(o.contextFunc), streamClientInterceptor(o.contextFunc)
}

func UnaryClientInterceptor(opts ...Option) grpc.UnaryClientInterceptor {
	o := defaultClientOptions()
	o.apply(opts...)
	return unaryClientInterceptor(o.contextFunc)
}

func StreamClientInterceptor(opts ...Option) grpc.StreamClientInterceptor {
	o := defaultClientOptions()
	o.apply(opts...)
	return streamClientInterceptor(o.contextFunc)
}

func unaryClientInterceptor(
	contextFunc ContextFunc,
) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req interface{},
		reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctx = contextFunc(ctx)
		err := invoker(ctx, method, req, reply, cc, opts...)
		return err
	}
}

func streamClientInterceptor(
	contextFunc ContextFunc,
) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		ctx = contextFunc(ctx)
		return streamer(ctx, desc, cc, method, opts...)
	}
}
