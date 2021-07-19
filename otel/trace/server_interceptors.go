package grpcoteltrace

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// ServerInterceptor returns new unary and stream server interceptors for otel trace.
func ServerInterceptor(opts ...Option) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	o := defaultClientOptions()
	o.apply(opts...)
	return unaryServerInterceptor(o.tracer, o.propagator, o.contextFunc),
		streamServerInterceptor(o.tracer, o.propagator, o.contextFunc)
}

// UnaryServerInterceptor returns a new unary server interceptor for otel trace.
func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	o := defaultClientOptions()
	o.apply(opts...)
	return unaryServerInterceptor(o.tracer, o.propagator, o.contextFunc)
}

// StreamServerInterceptor returns a new streaming server interceptor for otel trace.
func StreamServerInterceptor(opts ...Option) grpc.StreamServerInterceptor {
	o := defaultClientOptions()
	o.apply(opts...)
	return streamServerInterceptor(o.tracer, o.propagator, o.contextFunc)
}

func unaryServerInterceptor(
	tracer trace.Tracer,
	propagator propagation.TextMapPropagator,
	contextFunc ContextFunc,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		ctx, span := startSpan(ctx, tracer, propagator, info.FullMethod, trace.SpanKindServer)
		ctx = contextFunc(ctx)
		resp, err := handler(ctx, req)
		endSpan(err, span)
		return resp, err
	}
}

func streamServerInterceptor(
	tracer trace.Tracer,
	propagator propagation.TextMapPropagator,
	streamHandlerFunc ContextFunc,
) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := stream.Context()
		ctx, span := startSpan(ctx, tracer, propagator, info.FullMethod, trace.SpanKindServer)
		ctx = streamHandlerFunc(ctx)
		streamWithTrace := &serverStreamWithTrace{ServerStream: stream, span: span, ctx: ctx}
		err := handler(srv, streamWithTrace)
		endSpan(err, span)
		return err
	}
}

type serverStreamWithTrace struct {
	grpc.ServerStream
	span            trace.Span
	ctx             context.Context
	mu              sync.Mutex
	alreadyFinished bool
}

func (w *serverStreamWithTrace) SetHeader(md metadata.MD) error {
	err := w.ServerStream.SetHeader(md)
	if err != nil {
		w.endSpan(err)
	}
	return err
}
func (w *serverStreamWithTrace) SendHeader(md metadata.MD) error {
	err := w.ServerStream.SetHeader(md)
	if err != nil {
		w.endSpan(err)
	}
	return err
}
func (w *serverStreamWithTrace) SetTrailer(md metadata.MD) {
	w.ServerStream.SetTrailer(md)
}

func (w *serverStreamWithTrace) Context() context.Context {
	return w.ctx
}

func (w *serverStreamWithTrace) SendMsg(m interface{}) error {
	err := w.ServerStream.SendMsg(m)
	if err != nil {
		w.endSpan(err)
	}
	return err
}
func (w *serverStreamWithTrace) RecvMsg(m interface{}) error {
	err := w.ServerStream.RecvMsg(m)
	if err != nil {
		w.endSpan(err)
	}
	return err
}

func (s *serverStreamWithTrace) endSpan(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.alreadyFinished {
		endSpan(err, s.span)
		s.alreadyFinished = true
	}
}
