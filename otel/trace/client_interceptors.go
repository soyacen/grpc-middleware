package grpcoteltrace

import (
	"context"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

func ClientInterceptor(opts ...Option) (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	o := defaultOptions()
	o.apply(opts...)
	return unaryClientInterceptor(o.tracer, o.propagator),
		streamClientInterceptor(o.tracer, o.propagator)
}

func UnaryClientInterceptor(opts ...Option) grpc.UnaryClientInterceptor {
	o := defaultOptions()
	o.apply(opts...)
	return unaryClientInterceptor(o.tracer, o.propagator)
}

func StreamClientInterceptor(opts ...Option) grpc.StreamClientInterceptor {
	o := defaultOptions()
	o.apply(opts...)
	return streamClientInterceptor(o.tracer, o.propagator)
}

func unaryClientInterceptor(
	tracer trace.Tracer,
	propagator propagation.TextMapPropagator,
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
		ctx, span := startSpan(ctx, tracer, propagator, method, trace.SpanKindClient)
		err := invoker(ctx, method, req, reply, cc, opts...)
		endSpan(err, span)
		return err
	}
}

func streamClientInterceptor(
	tracer trace.Tracer,
	propagator propagation.TextMapPropagator,
) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		ctx, span := startSpan(ctx, tracer, propagator, method, trace.SpanKindClient)
		clientStream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			endSpan(err, span)
			return nil, err
		}
		return clientStream, nil
	}
}
