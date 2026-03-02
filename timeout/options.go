package timeout

import (
	"time"
)

// options 存储超时的配置选项
type options struct {
	// Timeout 超时时间，超过此时间的请求会被记录
	Timeout time.Duration
}

// defaultOptions 创建默认的慢请求日志选项
// 默认阈值为5秒
func defaultOptions() *options {
	return &options{
		Timeout: 5 * time.Second,
	}
}

// apply 应用所有配置选项
func (o *options) apply(opts ...Option) *options {
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Option 表示慢请求日志的配置选项函数
type Option func(o *options)

// Timeout 设置超时时间
//
// 参数:
//   - timeout: 超时时间
//
// 返回值:
//   - Option: 配置选项函数
func Timeout(timeout time.Duration) Option {
	return func(o *options) {
		o.Timeout = timeout
	}
}
