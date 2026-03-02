// Package context 提供gRPC上下文处理中间件功能
// 用于在gRPC请求处理过程中修改和增强上下文
package context

import (
	"context"

	"google.golang.org/grpc"
)

// UnaryServerInterceptor 创建一元服务器上下文拦截器
// 该拦截器在处理一元gRPC请求时可以修改请求上下文
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
		// 使用配置的上下文函数处理上下文
		ctx = o.contextFunc(ctx)
		// 继续处理请求
		resp, err := handler(ctx, req)
		return resp, err
	}
}

// StreamServerInterceptor 创建流式服务器上下文拦截器
// 该拦截器在处理流式gRPC请求时可以修改请求上下文
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
		// 获取流的上下文并进行处理
		ctx := stream.Context()
		ctx = o.contextFunc(ctx)
		// 继续处理请求
		return handler(srv, stream)
	}
}

// UnaryClientInterceptor 创建一元客户端上下文拦截器
// 该拦截器在发起一元gRPC客户端调用时可以修改请求上下文
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
		// 使用配置的上下文函数处理上下文
		ctx = o.contextFunc(ctx)
		// 发起gRPC调用
		err := invoker(ctx, method, req, reply, cc, opts...)
		return err
	}
}

// StreamClientInterceptor 创建流式客户端上下文拦截器
// 该拦截器在发起流式gRPC客户端调用时可以修改请求上下文
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
		// 使用配置的上下文函数处理上下文
		ctx = o.contextFunc(ctx)
		// 发起流式gRPC调用
		return streamer(ctx, desc, cc, method, opts...)
	}
}
