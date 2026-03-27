// Package slowlog 提供gRPC慢请求日志记录中间件功能
// 用于监控和记录超过阈值的慢速gRPC请求
package slowlog

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
)

// UnaryServerInterceptor 创建一元服务器慢请求日志拦截器
// 该拦截器监控一元gRPC请求的执行时间，记录超过阈值的慢请求
//
// 参数:
//   - opts: 可选的配置选项
//
// 返回值:
//   - grpc.UnaryServerInterceptor: gRPC一元服务器拦截器
func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	// 获取默认选项并应用用户配置
	o := defaultOptions()
	o.apply(opts...)
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// 使用defer在函数结束时检查执行时间
		defer func(startTime time.Time) {
			elapsed := time.Since(startTime)
			// 如果执行时间超过阈值，记录慢请求日志
			if elapsed > o.SlowRequestThreshold {
				logSlowRequest(ctx, info.FullMethod, elapsed)
			}
		}(time.Now())
		// 执行原始处理器
		return handler(ctx, req)
	}
}

// StreamServerInterceptor 创建流式服务器慢请求日志拦截器
// 该拦截器监控流式gRPC请求的执行时间，记录超过阈值的慢请求
//
// 参数:
//   - opts: 可选的配置选项
//
// 返回值:
//   - grpc.StreamServerInterceptor: gRPC流式服务器拦截器
func StreamServerInterceptor(opts ...Option) grpc.StreamServerInterceptor {
	// 获取默认选项并应用用户配置
	o := defaultOptions()
	o.apply(opts...)
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// 获取流的上下文
		ctx := stream.Context()
		// 使用defer在函数结束时检查执行时间
		defer func(startTime time.Time) {
			elapsed := time.Since(startTime)
			// 如果执行时间超过阈值，记录慢请求日志
			if elapsed > o.SlowRequestThreshold {
				logSlowRequest(ctx, info.FullMethod, elapsed)
			}
		}(time.Now())
		// 执行原始处理器
		return handler(srv, stream)
	}
}

// UnaryClientInterceptor 创建一元客户端慢请求日志拦截器
// 该拦截器监控一元gRPC客户端调用的执行时间，记录超过阈值的慢请求
//
// 参数:
//   - opts: 可选的配置选项
//
// 返回值:
//   - grpc.UnaryClientInterceptor: gRPC一元客户端拦截器
func UnaryClientInterceptor(opts ...Option) grpc.UnaryClientInterceptor {
	// 获取默认选项并应用用户配置
	o := defaultOptions()
	o.apply(opts...)
	return func(
		ctx context.Context,
		method string,
		req interface{},
		reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// 使用defer在函数结束时检查执行时间
		defer func(startTime time.Time) {
			elapsed := time.Since(startTime)
			// 如果执行时间超过阈值，记录慢请求日志
			if elapsed > o.SlowRequestThreshold {
				logSlowRequest(ctx, method, elapsed)
			}
		}(time.Now())
		// 执行gRPC调用
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// StreamClientInterceptor 创建流式客户端慢请求日志拦截器
// 该拦截器监控流式gRPC客户端调用的执行时间，记录超过阈值的慢请求
//
// 参数:
//   - opts: 可选的配置选项
//
// 返回值:
//   - grpc.StreamClientInterceptor: gRPC流式客户端拦截器
func StreamClientInterceptor(opts ...Option) grpc.StreamClientInterceptor {
	// 获取默认选项并应用用户配置
	o := defaultOptions()
	o.apply(opts...)
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		// 使用defer在函数结束时检查执行时间
		defer func(startTime time.Time) {
			elapsed := time.Since(startTime)
			// 如果执行时间超过阈值，记录慢请求日志
			if elapsed > o.SlowRequestThreshold {
				logSlowRequest(ctx, method, elapsed)
			}
		}(time.Now())
		// 执行流式gRPC调用
		return streamer(ctx, desc, cc, method, opts...)
	}
}

// logSlowRequest 记录慢请求日志
//
// 参数:
//   - ctx: 请求上下文
//   - method: 方法名
//   - elapsed: 执行耗时
func logSlowRequest(ctx context.Context, method string, elapsed time.Duration) {
	slog.WarnContext(ctx, "Slow gRPC call", slog.String("duration", elapsed.String()), slog.String("method", method))
}
