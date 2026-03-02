// Package slowlog 提供gRPC慢请求日志的配置选项
package slowlog

import (
	"time"
)

// options 存储慢请求日志的配置选项
type options struct {
	// SlowRequestThreshold 慢请求阈值，超过此时间的请求会被记录
	SlowRequestThreshold time.Duration
}

// apply 应用所有配置选项
func (o *options) apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

// Option 表示慢请求日志的配置选项函数
type Option func(o *options)

// defaultOptions 创建默认的慢请求日志选项
// 默认阈值为5秒
func defaultOptions() *options {
	return &options{
		SlowRequestThreshold: 5 * time.Second,
	}
}

// SlowRequestThreshold 设置慢请求阈值
//
// 参数:
//   - threshold: 慢请求的时间阈值
//
// 返回值:
//   - Option: 配置选项函数
func SlowRequestThreshold(threshold time.Duration) Option {
	return func(o *options) {
		o.SlowRequestThreshold = threshold
	}
}
