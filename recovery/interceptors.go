// Package recovery 提供gRPC恢复中间件功能
// 用于捕获和处理gRPC服务中的panic异常，防止服务崩溃
package recovery

import (
	"context"
	"fmt"
	"runtime/debug"

	"google.golang.org/grpc"
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

// UnaryServerInterceptor 创建一元服务器恢复拦截器
// 该拦截器捕获一元gRPC请求处理过程中的panic异常
//
// 参数:
//   - opts: 可选的配置选项
//
// 返回值:
//   - grpc.UnaryServerInterceptor: gRPC一元服务器拦截器
func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	// 应用配置选项
	opt := defaultOptions().apply(opts...)
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ any, err error) {
		// 使用defer捕获可能发生的panic
		defer func() {
			p := recover()
			if p == nil {
				return
			}
			// 使用配置的处理函数处理panic
			err = opt.handler(ctx, info.FullMethod, p)
		}()

		// 执行原始处理器
		return handler(ctx, req)
	}
}

// StreamServerInterceptor 创建流式服务器恢复拦截器
// 该拦截器捕获流式gRPC请求处理过程中的panic异常
//
// 参数:
//   - opts: 可选的配置选项
//
// 返回值:
//   - grpc.StreamServerInterceptor: gRPC流式服务器拦截器
func StreamServerInterceptor(opts ...Option) grpc.StreamServerInterceptor {
	// 应用配置选项
	opt := defaultOptions().apply(opts...)
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		// 使用defer捕获可能发生的panic
		defer func() {
			p := recover()
			if p == nil {
				return
			}
			// 使用配置的处理函数处理panic
			err = opt.handler(stream.Context(), info.FullMethod, p)
		}()

		// 执行原始处理器
		return handler(srv, stream)
	}
}

// PanicError 表示捕获到的panic错误
// 包含方法名、panic值和堆栈跟踪信息
type PanicError struct {
	// Method 发生panic的方法名
	Method string
	// Panic panic的值
	Panic any
	// Stack 堆栈跟踪信息
	Stack []byte
}

// Error 实现error接口，返回格式化的错误信息
func (e *PanicError) Error() string {
	return fmt.Sprintf("panic caught: %s\n\n%v\n\n%s", e.Method, e.Panic, e.Stack)
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
