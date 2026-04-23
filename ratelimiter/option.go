package ratelimiter

import (
	"context"
	"time"
)

// options 限流器配置选项
type options struct {
	// Window 统计时间窗口
	Window time.Duration

	// Buckets 时间窗口内的桶数量
	Buckets int

	// CPUThreshold CPU使用率阈值（0.0-1.0）
	CPUThreshold float64

	// CPU 获取CPU使用率的函数
	CPU func() float64

	// CPUInterval CPU采样间隔
	CPUInterval time.Duration

	// Skip 跳过限流的判断函数，返回true时跳过限流
	Skip func(ctx context.Context, fullMethod string) bool

	rateLimiter RateLimiter
}

// Option 配置选项函数类型
type Option func(*options)

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

// WithCPUThreshold 设置CPU使用率阈值
func WithCPUThreshold(threshold float64) Option {
	return func(o *options) {
		o.CPUThreshold = threshold
	}
}

// WithCPU 设置CPU使用率获取函数
func WithCPU(cpu func() float64) Option {
	return func(o *options) {
		o.CPU = cpu
	}
}

// WithCPUInterval 设置CPU采样间隔
func WithCPUInterval(interval time.Duration) Option {
	return func(o *options) {
		o.CPUInterval = interval
	}
}

// WithSkip 设置跳过限流的判断函数
func WithSkip(skip func(ctx context.Context, fullMethod string) bool) Option {
	return func(o *options) {
		o.Skip = skip
	}
}

// withRateLimiter 设置自定义限流器（用于测试）
func withRateLimiter(limiter RateLimiter) Option {
	return func(o *options) {
		o.rateLimiter = limiter
	}
}

// defaultOptions 返回默认配置
func defaultOptions() *options {
	return &options{
		Window:       time.Second * 10,
		Buckets:      100,
		CPUThreshold: 0.8,
		CPU:          defaultCPU,
		CPUInterval:  time.Millisecond * 500,
	}
}

// init 初始化配置参数
func (o *options) init() *options {
	if o.Window <= 0 {
		o.Window = time.Second * 10
	}
	if o.Buckets <= 0 {
		o.Buckets = 100
	}
	if o.CPUThreshold <= 0 {
		o.CPUThreshold = 0.8
	}
	if o.CPUInterval <= 0 {
		o.CPUInterval = time.Millisecond * 500
	}
	if o.CPU == nil {
		o.CPU = defaultCPU
		setCPUInterval(o.CPUInterval)
	}
	return o
}

// apply 应用配置选项
func (o *options) apply(opts ...Option) *options {
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// newRateLimiter 创建限流器实例
func (o *options) newRateLimiter() RateLimiter {
	if o.rateLimiter != nil {
		return o.rateLimiter
	}
	return &bbrRateLimiter{
		conf:     o,
		passStat: newRollingCounter(o.Window, o.Buckets, false),
		rtStat:   newRollingCounter(o.Window, o.Buckets, true),
		cpu:      o.CPU,
	}
}
