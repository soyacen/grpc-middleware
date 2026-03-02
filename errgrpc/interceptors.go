// Package er r 提供gRPC错误处理中间件功能
// 用于统一处理和转换gRPC服务端的错误响应
package err

import (
	"context"

	"google.golang.org/grpc"
)

// UnaryServerInterceptor 创建一元服务器错误处理拦截器
// 该拦截器在处理一元gRPC请求后统一处理错误响应
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
		// 执行原始处理器
		resp, err := handler(ctx, req)
		// 使用配置的错误处理函数转换错误
		status := o.errorFunc(err)
		if status != nil {
			// 如果有状态信息，返回转换后的错误
			return nil, status.Err()
		}
		// 否则返回正常响应
		return resp, nil
	}
}

// StreamServerInterceptor 创建流式服务器错误处理拦截器
// 该拦截器在处理流式gRPC请求后统一处理错误响应
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
		// 执行原始处理器
		err := handler(srv, stream)
		// 使用配置的错误处理函数转换错误
		status := o.errorFunc(err)
		if status != nil {
			// 如果有状态信息，返回转换后的错误
			return status.Err()
		}
		// 否则返回无错误
		return nil
	}
}
