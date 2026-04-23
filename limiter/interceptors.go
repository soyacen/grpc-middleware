package limiter

import (
	"context"
	"fmt"

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
	o := defaultOptions().apply(opts...).init()
	limiter := o.newLimiter()

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if o.Skip != nil && o.Skip() {
			return handler(ctx, req)
		}

		done, err := limiter.Allow()
		if err != nil {
			return nil, err
		}

		defer func() {
			if r := recover(); r != nil {
				done(DoneInfo{Err: fmt.Errorf("panic: %v", r)})
				panic(r)
			}
		}()
		resp, err := handler(ctx, req)
		done(DoneInfo{Err: err})
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
	o := defaultOptions().apply(opts...).init()
	limiter := o.newLimiter()

	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if o.Skip != nil && o.Skip() {
			return handler(srv, stream)
		}

		done, err := limiter.Allow()
		if err != nil {
			return err
		}

		defer func() {
			if r := recover(); r != nil {
				done(DoneInfo{Err: fmt.Errorf("panic: %v", r)})
				panic(r)
			}
		}()
		err = handler(srv, stream)
		done(DoneInfo{Err: err})
		return err
	}
}
