package grpchystrix

import (
	"context"
	"time"

	"github.com/afex/hystrix-go/plugins"
)

// options is the hystrix client implementation
type options struct {
	timeout                time.Duration
	maxConcurrentRequests  int
	requestVolumeThreshold int
	sleepWindow            int
	errorPercentThreshold  int
	fallbackFunc           func(ctx context.Context, err error) error
	statsD                 *plugins.StatsdCollectorConfig
}

func (o *options) apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

// Option represents the hystrix client options
type Option func(*options)

// WithTimeout sets hystrix timeout
func WithTimeout(timeout time.Duration) Option {
	return func(c *options) {
		c.timeout = timeout
	}
}

// WithMaxConcurrentRequests sets hystrix max concurrent requests
func WithMaxConcurrentRequests(maxConcurrentRequests int) Option {
	return func(c *options) {
		c.maxConcurrentRequests = maxConcurrentRequests
	}
}

// WithRequestVolumeThreshold sets hystrix request volume threshold
func WithRequestVolumeThreshold(requestVolumeThreshold int) Option {
	return func(c *options) {
		c.requestVolumeThreshold = requestVolumeThreshold
	}
}

// WithSleepWindow sets hystrix sleep window
func WithSleepWindow(sleepWindow int) Option {
	return func(c *options) {
		c.sleepWindow = sleepWindow
	}
}

// WithErrorPercentThreshold sets hystrix error percent threshold
func WithErrorPercentThreshold(errorPercentThreshold int) Option {
	return func(c *options) {
		c.errorPercentThreshold = errorPercentThreshold
	}
}

// WithFallbackFunc sets the fallback function
func WithFallbackFunc(fn func(ctx context.Context, err error) error) Option {
	return func(c *options) {
		c.fallbackFunc = fn
	}
}

// WithStatsDCollector exports hystrix metrics to a statsD backend
func WithStatsDCollector(addr, prefix string, sampleRate float32, flushBytes int) Option {
	return func(c *options) {
		c.statsD = &plugins.StatsdCollectorConfig{StatsdAddr: addr, Prefix: prefix, SampleRate: sampleRate, FlushBytes: flushBytes}
	}
}

const (
	defaultHystrixTimeout         = 30 * time.Second
	defaultMaxConcurrentRequests  = 100
	defaultErrorPercentThreshold  = 25
	defaultSleepWindow            = 10
	defaultRequestVolumeThreshold = 10

	maxUint = ^uint(0)
	maxInt  = int(maxUint >> 1)
)

func defaultOptions() *options {
	return &options{
		fallbackFunc:           nil,
		timeout:                defaultHystrixTimeout,
		maxConcurrentRequests:  defaultMaxConcurrentRequests,
		errorPercentThreshold:  defaultErrorPercentThreshold,
		sleepWindow:            defaultSleepWindow,
		requestVolumeThreshold: defaultRequestVolumeThreshold,
	}
}
