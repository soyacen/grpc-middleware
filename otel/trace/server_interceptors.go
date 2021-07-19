package grpcoteltrace

import (
	"context"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

func ServerInterceptor(opts ...Option) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	o := defaultOptions()
	o.apply(opts...)
	return unaryServerInterceptor(o.tracer, o.propagator),
		streamServerInterceptor(o.tracer, o.propagator)
}

func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	o := defaultOptions()
	o.apply(opts...)
	return unaryServerInterceptor(o.tracer, o.propagator)
}

func StreamServerInterceptor(opts ...Option) grpc.StreamServerInterceptor {
	o := defaultOptions()
	o.apply(opts...)
	return streamServerInterceptor(o.tracer, o.propagator)
}

func unaryServerInterceptor(
	tracer trace.Tracer,
	propagator propagation.TextMapPropagator,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		ctx, span := startSpan(ctx, tracer, propagator, info.FullMethod, trace.SpanKindServer)
		resp, err := handler(ctx, req)
		endSpan(err, span)
		return resp, err
	}
}

func streamServerInterceptor(
	tracer trace.Tracer,
	propagator propagation.TextMapPropagator,
) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := stream.Context()
		ctx, span := startSpan(ctx, tracer, propagator, info.FullMethod, trace.SpanKindServer)
		err := handler(srv, stream)
		endSpan(err, span)
		return err
	}
}
