// Package errorlog 提供gRPC错误日志记录中间件功能
// 用于记录发生错误的gRPC请求，支持配置是否打印请求和响应
package errorlog

// options 存储错误日志的配置选项
type options struct {
	// PrintRequest 是否打印请求内容
	PrintRequest bool
	// PrintResponse 是否打印响应内容
	PrintResponse bool
}

// apply 应用所有配置选项
func (o *options) apply(opts ...Option) *options {
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Option 表示错误日志的配置选项函数
type Option func(o *options)

// defaultOptions 创建默认的错误日志选项
// 默认不打印请求和响应
func defaultOptions() *options {
	return &options{
		PrintRequest:  false,
		PrintResponse: false,
	}
}

// WithPrintRequest 设置是否打印请求内容
//
// 参数:
//   - enable: 是否启用请求打印
//
// 返回值:
//   - Option: 配置选项函数
func WithPrintRequest(enable bool) Option {
	return func(o *options) {
		o.PrintRequest = enable
	}
}

// WithPrintResponse 设置是否打印响应内容
//
// 参数:
//   - enable: 是否启用响应打印
//
// 返回值:
//   - Option: 配置选项函数
func WithPrintResponse(enable bool) Option {
	return func(o *options) {
		o.PrintResponse = enable
	}
}
