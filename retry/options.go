package retry

import (
	"time"

	"github.com/soyacen/gox/backoff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type options struct {
	MaxRetries     int
	BackoffFunc    backoff.Func
	RetryableFunc  func(error) bool
	PerCallTimeout time.Duration
}

func defaultOptions() *options {
	o := &options{
		MaxRetries:     3,
		BackoffFunc:    backoff.Exponential2(100 * time.Millisecond),
		RetryableFunc:  defaultRetryableFunc,
		PerCallTimeout: 0,
	}
	return o
}

func defaultRetryableFunc(err error) bool {
	if err == nil {
		return false
	}
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	switch st.Code() {
	case codes.Unavailable,
		codes.ResourceExhausted,
		codes.DeadlineExceeded,
		codes.Aborted:
		return true
	default:
		return false
	}
}

func (o *options) apply(opts ...Option) *options {
	for _, opt := range opts {
		opt(o)
	}
	return o
}

type Option func(o *options)

func MaxRetries(n int) Option {
	return func(o *options) {
		if n >= 0 {
			o.MaxRetries = n
		}
	}
}

func BackoffFunc(fn backoff.Func) Option {
	return func(o *options) {
		if fn != nil {
			o.BackoffFunc = fn
		}
	}
}

func WithRetryableFunc(fn func(error) bool) Option {
	return func(o *options) {
		if fn != nil {
			o.RetryableFunc = fn
		}
	}
}

func PerCallTimeout(d time.Duration) Option {
	return func(o *options) {
		o.PerCallTimeout = d
	}
}
