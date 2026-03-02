// Package context 提供gRPC上下文处理的配置选项
package context

import (
	"context"
)

// ContextFunc 定义上下文处理函数类型
// 用于自定义上下文的修改逻辑
type ContextFunc func(ctx context.Context) context.Context

// options 存储上下文处理的配置选项
type options struct {
	// contextFunc 上下文处理函数
	contextFunc ContextFunc
}

// apply 应用所有配置选项
func (o *options) apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

// Option 表示上下文处理的配置选项函数
type Option func(o *options)

// defaultOptions 创建默认的上下文处理选项
// 默认情况下不修改上下文，直接返回原上下文
func defaultOptions() *options {
	return &options{
		contextFunc: func(ctx context.Context) context.Context { return ctx },
	}
}

// WithContextFunc 设置自定义的上下文处理函数
//
// 参数:
//   - contextFunc: 自定义的上下文处理函数
//
// 返回值:
//   - Option: 配置选项函数
func WithContextFunc(contextFunc ContextFunc) Option {
	return func(o *options) {
		o.contextFunc = contextFunc
	}
}
