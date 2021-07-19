package grpccontext

import (
	"context"
)

type ContextFunc func(ctx context.Context) context.Context

type options struct {
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
		contextFunc: func(ctx context.Context) context.Context { return ctx },
	}
}

func WithContextFunc(contextFunc ContextFunc) Option {
	return func(o *options) {
		o.contextFunc = contextFunc
	}
}
