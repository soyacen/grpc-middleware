package grpcoteltrace

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type ContextFunc func(ctx context.Context) context.Context

type options struct {
	tracer      trace.Tracer
	propagator  propagation.TextMapPropagator
	contextFunc ContextFunc
}

func (o *options) apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

type Option func(o *options)

func defaultClientOptions() *options {
	return &options{
		tracer:      otel.Tracer(""),
		propagator:  otel.GetTextMapPropagator(),
		contextFunc: func(ctx context.Context) context.Context { return ctx },
	}
}

func WithTracer(tracer trace.Tracer) Option {
	return func(o *options) {
		o.tracer = tracer
	}
}

func WithPropagator(propagator propagation.TextMapPropagator) Option {
	return func(o *options) {
		o.propagator = propagator
	}
}

func WithContextFunc(contextFunc ContextFunc) Option {
	return func(o *options) {
		o.contextFunc = contextFunc
	}
}
