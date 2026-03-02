package limiter

import (
	"context"

	"google.golang.org/grpc"
)

// UnaryServerInterceptor 创建基于BBR算法的gRPC服务端一元调用拦截器
// 该拦截器在每个一元RPC调用前检查是否允许执行，实现自适应限流
//
// 参数:
//   - opts: 可选的配置选项，用于自定义限流器行为
//
// 返回:
//   - grpc.UnaryServerInterceptor: gRPC服务端一元调用拦截器函数
//
// 工作原理:
// 1. 在每次请求到达时调用 limiter.Allow() 检查是否允许执行
// 2. 如果不允许执行，直接返回限流错误
// 3. 如果允许执行，继续处理请求
// 4. 请求完成后调用 done() 回调函数更新统计信息
func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	// 初始化限流器配置并创建限流器实例
	limiter := defaultOptions().apply(opts...).init().newLimiter()

	return func(
		ctx context.Context, // 请求上下文
		req interface{}, // 请求参数
		info *grpc.UnaryServerInfo, // 服务端信息，包含方法名等
		handler grpc.UnaryHandler, // 实际的请求处理函数
	) (interface{}, error) {
		// 检查限流器是否允许执行本次请求
		// done是完成回调函数，err是限流错误（如果被限流）
		done, err := limiter.Allow()
		if err != nil {
			// 请求被限流，直接返回错误
			return nil, err
		}

		// 执行实际的请求处理逻辑
		resp, err := handler(ctx, req)

		// 调用完成回调函数，传入处理结果
		// 限流器会根据处理结果更新内部统计信息
		done(DoneInfo{Err: err})

		// 返回处理结果给客户端
		return resp, err
	}
}

// StreamServerInterceptor 创建基于BBR算法的gRPC服务端流式调用拦截器
// 该拦截器在每个流式RPC调用前检查是否允许执行，实现自适应限流
//
// 参数:
//   - opts: 可选的配置选项，用于自定义限流器行为
//
// 返回:
//   - grpc.StreamServerInterceptor: gRPC服务端流式调用拦截器函数
//
// 工作原理:
// 1. 在流式调用建立时检查是否允许执行
// 2. 如果不允许执行，直接返回限流错误
// 3. 如果允许执行，建立流式连接
// 4. 流式调用结束时调用 done() 回调函数更新统计信息
func StreamServerInterceptor(opts ...Option) grpc.StreamServerInterceptor {
	// 初始化限流器配置并创建限流器实例
	limiter := defaultOptions().apply(opts...).init().newLimiter()

	return func(
		srv interface{}, // 服务实现
		stream grpc.ServerStream, // 流式连接
		info *grpc.StreamServerInfo, // 流式服务信息
		handler grpc.StreamHandler, // 实际的流式处理函数
	) error {
		// 检查限流器是否允许执行本次流式调用
		done, err := limiter.Allow()
		if err != nil {
			// 流式调用被限流，直接返回错误
			return err
		}

		// 执行实际的流式处理逻辑
		err = handler(srv, stream)

		// 调用完成回调函数，传入处理结果
		// 限流器会根据处理结果更新内部统计信息
		done(DoneInfo{Err: err})

		// 返回处理结果
		return err
	}
}
