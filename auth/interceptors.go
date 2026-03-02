// Package auth 提供gRPC认证中间件功能
package auth

import (
	"context"

	"google.golang.org/grpc"
)

// AuthFunc 定义认证函数类型，用于验证请求并返回新的上下文
// 参数:
//   - ctx: 请求上下文
//   - fullMethodName: 完整的方法名
//
// 返回值:
//   - context.Context: 认证后的上下文
//   - error: 认证错误
type AuthFunc func(ctx context.Context, fullMethodName string) (context.Context, error)

// UnaryServerInterceptor 创建一元服务器拦截器
// 该拦截器在处理每个一元gRPC请求前执行认证逻辑
//
// 参数:
//   - authFunc: 认证函数
//
// 返回值:
//   - grpc.UnaryServerInterceptor: gRPC一元服务器拦截器
func UnaryServerInterceptor(authFunc AuthFunc) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// 执行认证逻辑，获取新的上下文
		newCtx, err := authFunc(ctx, info.FullMethod)
		if err != nil {
			return nil, err
		}
		// 使用认证后的上下文继续处理请求
		return handler(newCtx, req)
	}
}

// StreamServerInterceptor 创建流式服务器拦截器
// 该拦截器在处理每个流式gRPC请求前执行认证逻辑
//
// 参数:
//   - authFunc: 认证函数
//
// 返回值:
//   - grpc.StreamServerInterceptor: gRPC流式服务器拦截器
func StreamServerInterceptor(authFunc AuthFunc) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// 执行认证逻辑，获取新的上下文
		newCtx, err := authFunc(stream.Context(), info.FullMethod)
		if err != nil {
			return err
		}
		// 包装流以注入认证后的上下文
		wrapped := WrapServerStream(stream)
		wrapped.WrappedContext = newCtx
		// 使用包装后的流继续处理请求
		return handler(srv, wrapped)
	}
}

// WrappedServerStream 是对grpc.ServerStream的包装结构体
// 用于在流式处理中注入自定义上下文
type WrappedServerStream struct {
	grpc.ServerStream
	// WrappedContext 存储包装后的上下文
	WrappedContext context.Context
}

// Context 返回包装后的上下文
// 实现grpc.ServerStream接口的Context方法
func (w *WrappedServerStream) Context() context.Context {
	return w.WrappedContext
}

// WrapServerStream 将普通的ServerStream包装为WrappedServerStream
// 如果stream已经是WrappedServerStream类型，则直接返回
// 否则创建新的包装实例
//
// 参数:
//   - stream: 原始的gRPC服务器流
//
// 返回值:
//   - *WrappedServerStream: 包装后的服务器流
func WrapServerStream(stream grpc.ServerStream) *WrappedServerStream {
	// 检查是否已经包装过
	if existing, ok := stream.(*WrappedServerStream); ok {
		return existing
	}
	// 创建新的包装实例
	return &WrappedServerStream{ServerStream: stream, WrappedContext: stream.Context()}
}
