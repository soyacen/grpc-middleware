package circuitbreaker

import (
	"time"

	"github.com/soyacen/gox/randx"
)

// options 熔断器配置选项
type options struct {
	K       float64
	Window  time.Duration
	Buckets int
}

// Option 配置选项函数类型
type Option func(*options)

// WithK 设置熔断因子
func WithK(k float64) Option {
	return func(o *options) {
		o.K = k
	}
}

// WithWindow 设置统计时间窗口
func WithWindow(window time.Duration) Option {
	return func(o *options) {
		o.Window = window
	}
}

// WithBuckets 设置时间窗口内的桶数量
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

func (o *options) newCircuitBreaker() CircuitBreaker {
	rnd, err := randx.NewPCG() // Create a new PCG generator if none available.
	if err != nil {
		panic(err) // Panic on failure to initialize due to crypto/rand issues.
	}
	return &sreCircuitBreaker{
		k:      o.K,
		window: newRollingCounter(o.Window, o.Buckets),
		rnd:    rnd,
	}
}
