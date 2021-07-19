package grpclogger

import (
	"context"
)

type Logger interface {
	Log(fields map[string]interface{})
}

type LoggerFactory func(ctx context.Context) Logger

type options struct {
	loggerFactory LoggerFactory
}

func (o *options) apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

type Option func(o *options)

func defaultOptions() *options {
	return &options{}
}

func WithLoggerFactory(loggerFactory LoggerFactory) Option {
	return func(o *options) {
		o.loggerFactory = loggerFactory
	}
}
