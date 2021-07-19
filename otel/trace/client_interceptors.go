package grpcoteltrace

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// ClientInterceptor returns unary client and stream interceptors and for otel trace.
func ClientInterceptor(opts ...Option) (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	o := defaultClientOptions()
	o.apply(opts...)
	return unaryClientInterceptor(o.tracer, o.propagator, o.contextFunc),
		streamClientInterceptor(o.tracer, o.propagator, o.contextFunc)
}

// UnaryClientInterceptor returns a new unary client interceptor for otel trace.
func UnaryClientInterceptor(opts ...Option) grpc.UnaryClientInterceptor {
	o := defaultClientOptions()
	o.apply(opts...)
	return unaryClientInterceptor(o.tracer, o.propagator, o.contextFunc)
}

// StreamClientInterceptor returns a new streaming client interceptor for otel trace.
func StreamClientInterceptor(opts ...Option) grpc.StreamClientInterceptor {
	o := defaultClientOptions()
	o.apply(opts...)
	return streamClientInterceptor(o.tracer, o.propagator, o.contextFunc)
}

func unaryClientInterceptor(
	tracer trace.Tracer,
	propagator propagation.TextMapPropagator,
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
		ctx, span := startSpan(ctx, tracer, propagator, method, trace.SpanKindClient)
		ctx = contextFunc(ctx)
		err := invoker(ctx, method, req, reply, cc, opts...)
		endSpan(err, span)
		return err
	}
}

func streamClientInterceptor(
	tracer trace.Tracer,
	propagator propagation.TextMapPropagator,
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
		ctx, span := startSpan(ctx, tracer, propagator, method, trace.SpanKindClient)
		ctx = contextFunc(ctx)
		clientStream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			endSpan(err, span)
			return nil, err
		}
		return &clientStreamWithTrace{ClientStream: clientStream, span: span}, nil
	}
}

type clientStreamWithTrace struct {
	grpc.ClientStream
	span         trace.Span
	mu           sync.Mutex
	alreadyEnded bool
}

func (s *clientStreamWithTrace) Header() (metadata.MD, error) {
	h, err := s.ClientStream.Header()
	if err != nil {
		s.endSpan(err)
	}
	return h, err
}

func (s *clientStreamWithTrace) SendMsg(m interface{}) error {
	err := s.ClientStream.SendMsg(m)
	if err != nil {
		s.endSpan(err)
	}
	return err
}

func (s *clientStreamWithTrace) CloseSend() error {
	err := s.ClientStream.CloseSend()
	s.endSpan(err)
	return err
}

func (s *clientStreamWithTrace) RecvMsg(m interface{}) error {
	err := s.ClientStream.RecvMsg(m)
	if err != nil {
		s.endSpan(err)
	}
	return err
}

func (s *clientStreamWithTrace) endSpan(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.alreadyEnded {
		endSpan(err, s.span)
		s.alreadyEnded = true
	}
}
