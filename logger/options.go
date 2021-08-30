package grpclogger

import (
	"context"
)

type Logger interface {
	Log(fields map[string]interface{})
}

type LoggerFactory func(ctx context.Context) Logger

type options struct {
	skip func(fullMethodName string, err error) bool
}

func (o *options) apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

type Option func(o *options)

func defaultOptions() *options {
	return &options{
		skip: func(fullMethodName string, err error) bool {
			return false
		},
	}
}

func WithSkip(skip func(fullMethodName string, err error) bool) Option {
	return func(o *options) {
		o.skip = skip
	}
}
