package recovery

import (
	"context"
	"runtime/debug"
)

// options 存储恢复中间件的配置选项
type options struct {
	// handler panic处理函数
	handler HandlerFunc
}

// Option 定义用于配置恢复中间件选项的函数类型
type Option func(*options)

// defaultOptions 返回默认的恢复选项
func defaultOptions() *options {
	return &options{
		handler: defaultHandler,
	}
}

// apply 将给定的选项应用到选项结构体中
func (o *options) apply(opts ...Option) *options {
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// HandlerFunc 定义panic处理函数类型
// 用于自定义panic发生时的处理逻辑
//
// 参数:
//   - ctx: 请求上下文
//   - method: 方法名
//   - p: panic的值
//
// 返回值:
//   - error: 处理后的错误
type HandlerFunc func(ctx context.Context, method string, p any) error

// RecoveryHandler 设置自定义的panic处理函数
//
// 参数:
//   - f: 自定义的panic处理函数
//
// 返回值:
//   - Option: 配置选项函数
func RecoveryHandler(f HandlerFunc) Option {
	return func(o *options) {
		o.handler = f
	}
}

// defaultHandler 默认的panic处理函数
// 创建包含完整信息的PanicError并返回
func defaultHandler(ctx context.Context, method string, p any) error {
	err := &PanicError{
		Method: method,
		Panic:  p,
		Stack:  debug.Stack(),
	}
	return err
}
