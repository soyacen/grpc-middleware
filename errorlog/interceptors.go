// Package errorlog 提供gRPC错误日志记录中间件功能
// 用于记录发生错误的gRPC请求，支持配置是否打印请求和响应
package errorlog

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor 创建一元服务器错误日志拦截器
// 该拦截器捕获一元gRPC请求的错误，并记录错误日志
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
		// 如果发生错误，记录错误日志
		if err != nil {
			logError(ctx, "unary", "server", info.FullMethod, err, req, resp, o)
		}
		return resp, err
	}
}

// StreamServerInterceptor 创建流式服务器错误日志拦截器
// 该拦截器捕获流式gRPC请求的错误，并记录错误日志
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
		// 执行原始处理器
		err := handler(srv, stream)
		// 如果发生错误，记录错误日志
		if err != nil {
			logError(ctx, "stream", "server", info.FullMethod, err, nil, nil, o)
		}
		return err
	}
}

// UnaryClientInterceptor 创建一元客户端错误日志拦截器
// 该拦截器捕获一元gRPC客户端调用的错误，并记录错误日志
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
		// 执行gRPC调用
		err := invoker(ctx, method, req, reply, cc, opts...)
		// 如果发生错误，记录错误日志
		if err != nil {
			logError(ctx, "unary", "client", method, err, req, reply, o)
		}
		return err
	}
}

// StreamClientInterceptor 创建流式客户端错误日志拦截器
// 该拦截器捕获流式gRPC客户端调用的错误，并记录错误日志
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
		// 执行流式gRPC调用
		stream, err := streamer(ctx, desc, cc, method, opts...)
		// 如果发生错误，记录错误日志
		if err != nil {
			logError(ctx, "stream", "client", method, err, nil, nil, o)
		}
		return stream, err
	}
}

// logError 记录错误日志
//
// 参数:
//   - ctx: 请求上下文
//   - rpcType: RPC类型（unary/stream）
//   - system: 系统（client/server）
//   - method: 方法名
//   - err: 错误对象
//   - req: 请求对象（可能为nil）
//   - resp: 响应对象（可能为nil）
//   - opts: 配置选项
func logError(ctx context.Context, rpcType string, system string, method string, err error, req interface{}, resp interface{}, opts *options) {
	// 获取gRPC状态码
	st, _ := status.FromError(err)

	// 构建日志属性
	attrs := []slog.Attr{
		slog.String("rpc_type", rpcType),
		slog.String("system", system),
		slog.String("method", method),
		slog.String("code", st.Code().String()),
	}
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}

	// 如果配置为打印请求，添加请求内容
	if opts.PrintRequest && req != nil {
		attrs = append(attrs, slog.Any("request", req))
	}

	// 如果配置为打印响应，添加响应内容
	if opts.PrintResponse && resp != nil {
		attrs = append(attrs, slog.Any("response", resp))
	}

	// 记录错误日志
	slog.LogAttrs(ctx, slog.LevelError, "gRPC call error", attrs...)
}
