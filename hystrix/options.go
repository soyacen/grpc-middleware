package grpchystrix

import (
	"time"

	"github.com/afex/hystrix-go/plugins"
)

type fallbackFunc func(error) error

// options is the hystrix client implementation
type options struct {
	hystrixTimeout         time.Duration
	maxConcurrentRequests  int
	requestVolumeThreshold int
	sleepWindow            int
	errorPercentThreshold  int
	fallbackFunc           func(err error) error
	statsD                 *plugins.StatsdCollectorConfig
}

func (o *options) apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

// Option represents the hystrix client options
type Option func(*options)

// WithHystrixTimeout sets hystrix timeout
func WithHystrixTimeout(timeout time.Duration) Option {
	return func(c *options) {
		c.hystrixTimeout = timeout
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
func WithFallbackFunc(fn fallbackFunc) Option {
	return func(c *options) {
		c.fallbackFunc = fn
	}
}

// WithStatsDCollector exports hystrix metrics to a statsD backend
func WithStatsDCollector(addr, prefix string) Option {
	return func(c *options) {
		c.statsD = &plugins.StatsdCollectorConfig{StatsdAddr: addr, Prefix: prefix}
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
		hystrixTimeout:         defaultHystrixTimeout,
		maxConcurrentRequests:  defaultMaxConcurrentRequests,
		errorPercentThreshold:  defaultErrorPercentThreshold,
		sleepWindow:            defaultSleepWindow,
		requestVolumeThreshold: defaultRequestVolumeThreshold,
	}
}
