package circuitbreaker

import (
	"math/rand"
	"time"
)

type options struct {
	K       float64
	Window  time.Duration
	Buckets int
}

type Option func(*options)

func WithK(k float64) Option {
	return func(o *options) {
		o.K = k
	}
}

func WithWindow(window time.Duration) Option {
	return func(o *options) {
		o.Window = window
	}
}

func WithBuckets(buckets int) Option {
	return func(o *options) {
		o.Buckets = buckets
	}
}

func defaultOptions() *options {
	return &options{
		K:       2.0,
		Window:  time.Second * 10,
		Buckets: 40,
	}
}

func (o *options) init() *options {
	if o.K <= 0 {
		o.K = 2.0
	}
	if o.Window <= 0 {
		o.Window = time.Second * 10
	}
	if o.Buckets <= 0 {
		o.Buckets = 40
	}
	return o
}

func (o *options) apply(opts ...Option) *options {
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func (o *options) newSREBreaker() SREBreaker {
	return &sreBreaker{
		k:      o.K,
		window: newWindow(o.Window, o.Buckets),
		rnd:    rndPool.Get().(*rand.Rand),
	}
}
