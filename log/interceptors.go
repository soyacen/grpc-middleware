// Package log 提供gRPC日志记录中间件功能
// 用于记录gRPC请求的访问日志，包括请求时间、延迟、状态码等信息
package log

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/soyacen/gox/slogx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	o := defaultOptions().apply(opts...)

	pool := sync.Pool{
		New: func() interface{} {
			fields := make([]slog.Attr, 0, 10)
			return &fields
		},
	}

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if o.loggerFactory == nil {
			return handler(ctx, req)
		}
		// Create a logger using the logger factory
		logger, err := o.loggerFactory(ctx)
		if err != nil {
			slog.Error("log: failed to get logger", slog.String("error", err.Error()))
			return handler(ctx, req)
		}
		startTime := time.Now()
		resp, err := handler(ctx, req)
		if o.skip(info.FullMethod, err) {
			return resp, err
		}
		fields := *pool.Get().(*[]slog.Attr)
		fields = append(fields,
			slog.String("system", "grpc.server"),
			slog.String("timestamp", startTime.Format(time.RFC3339)),
			slog.String("latency", time.Since(startTime).String()),
			slogx.Uint("status", status.Code(err)),
			slogx.Error("error", err),
		)
		if peer, ok := peer.FromContext(ctx); ok {
			fields = append(fields, slog.String("peer", peer.Addr.String()))
		}
		if d, ok := ctx.Deadline(); ok {
			fields = append(fields, slog.String("deadline", d.Format(time.RFC3339)))
		}
		logger.LogAttrs(ctx, o.level, info.FullMethod, fields...)
		// Reset the slice length to 0 to reuse the underlying array
		fields = fields[:0]
		// Put the slice back into the pool for reuse
		pool.Put(&fields)
		return resp, err
	}
}

// StreamServerInterceptor 创建流式服务器日志拦截器
// 该拦截器记录流式gRPC请求的详细访问日志
//
// 参数:
//   - opts: 可选的配置选项
//
// 返回值:
//   - grpc.StreamServerInterceptor: gRPC流式服务器拦截器
func StreamServerInterceptor(opts ...Option) grpc.StreamServerInterceptor {
	// 应用配置选项
	o := defaultOptions().apply(opts...)

	// 创建同步池用于复用日志字段切片，提高性能
	pool := sync.Pool{
		New: func() interface{} {
			fields := make([]slog.Attr, 0, 20)
			return &fields
		},
	}

	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// 如果没有配置日志工厂，直接执行处理器
		if o.loggerFactory == nil {
			return handler(srv, stream)
		}
		// 获取流的上下文
		ctx := stream.Context()
		// 使用日志工厂创建日志记录器
		logger, err := o.loggerFactory(ctx)
		if err != nil {
			slog.Error("log: failed to get logger", slog.String("error", err.Error()))
			return handler(srv, stream)
		}
		// 记录开始时间
		startTime := time.Now()
		// 执行原始处理器
		err = handler(srv, stream)
		// 检查是否需要跳过日志记录
		if o.skip(info.FullMethod, err) {
			return err
		}
		// 从池中获取字段切片
		fields := *pool.Get().(*[]slog.Attr)
		// 添加基本日志字段
		fields = append(fields,
			slog.String("system", "grpc.server"),
			slog.String("timestamp", startTime.Format(time.RFC3339)),
			slog.String("latency", time.Since(startTime).String()),
			slogx.Uint("status", status.Code(err)),
			slogx.Error("error", err),
		)
		// 添加客户端地址信息
		if peer, ok := peer.FromContext(ctx); ok {
			fields = append(fields, slog.String("peer", peer.Addr.String()))
		}
		// 添加截止时间信息
		if d, ok := ctx.Deadline(); ok {
			fields = append(fields, slog.String("deadline", d.Format(time.RFC3339)))
		}
		// 记录日志
		logger.LogAttrs(ctx, o.level, info.FullMethod, fields...)
		// 重置切片长度以便复用
		fields = fields[:0]
		// 将切片放回池中
		pool.Put(&fields)
		return err
	}
}

// UnaryClientInterceptor 创建一元客户端日志拦截器
// 该拦截器记录一元gRPC客户端调用的详细访问日志
//
// 参数:
//   - opts: 可选的配置选项
//
// 返回值:
//   - grpc.UnaryClientInterceptor: gRPC一元客户端拦截器
func UnaryClientInterceptor(opts ...Option) grpc.UnaryClientInterceptor {
	// 应用配置选项
	o := defaultOptions().apply(opts...)

	// 创建同步池用于复用日志字段切片，提高性能
	pool := sync.Pool{
		New: func() interface{} {
			fields := make([]slog.Attr, 0, 20)
			return &fields
		},
	}

	return func(
		ctx context.Context,
		method string,
		req interface{},
		reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// 如果没有配置日志工厂，直接执行调用
		if o.loggerFactory == nil {
			return invoker(ctx, method, req, reply, cc, opts...)
		}
		// 使用日志工厂创建日志记录器
		logger, err := o.loggerFactory(ctx)
		if err != nil {
			slog.Error("log: failed to get logger", slog.String("error", err.Error()))
			return invoker(ctx, method, req, reply, cc, opts...)
		}
		// 记录开始时间
		startTime := time.Now()
		// 执行gRPC调用
		err = invoker(ctx, method, req, reply, cc, opts...)
		// 检查是否需要跳过日志记录
		if o.skip(method, err) {
			return err
		}
		// 从池中获取字段切片
		fields := *pool.Get().(*[]slog.Attr)
		// 添加基本日志字段
		fields = append(fields,
			slog.String("system", "grpc.client"),
			slog.String("timestamp", startTime.Format(time.RFC3339)),
			slog.String("latency", time.Since(startTime).String()),
			slogx.Uint("status", status.Code(err)),
			slogx.Error("error", err),
		)
		// 添加服务端地址信息
		if peer, ok := peer.FromContext(ctx); ok {
			fields = append(fields, slog.String("peer", peer.Addr.String()))
		}
		// 添加截止时间信息
		if d, ok := ctx.Deadline(); ok {
			fields = append(fields, slog.String("deadline", d.Format(time.RFC3339)))
		}
		// 记录日志
		logger.LogAttrs(ctx, o.level, method, fields...)
		// 重置切片长度以便复用
		fields = fields[:0]
		// 将切片放回池中
		pool.Put(&fields)
		return err
	}
}

// StreamClientInterceptor 创建流式客户端日志拦截器
// 该拦截器记录流式gRPC客户端调用的详细访问日志
//
// 参数:
//   - opts: 可选的配置选项
//
// 返回值:
//   - grpc.StreamClientInterceptor: gRPC流式客户端拦截器
func StreamClientInterceptor(opts ...Option) grpc.StreamClientInterceptor {
	// 应用配置选项
	o := defaultOptions().apply(opts...)

	// 创建同步池用于复用日志字段切片，提高性能
	pool := sync.Pool{
		New: func() interface{} {
			fields := make([]slog.Attr, 0, 20)
			return &fields
		},
	}

	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		// 如果没有配置日志工厂，直接执行流式调用
		if o.loggerFactory == nil {
			return streamer(ctx, desc, cc, method, opts...)
		}
		// 使用日志工厂创建日志记录器
		logger, err := o.loggerFactory(ctx)
		if err != nil {
			slog.Error("log: failed to get logger", slog.String("error", err.Error()))
			return streamer(ctx, desc, cc, method, opts...)
		}
		// 记录开始时间
		startTime := time.Now()
		// 执行流式gRPC调用
		clientStream, err := streamer(ctx, desc, cc, method, opts...)
		// 检查是否需要跳过日志记录
		if o.skip(method, err) {
			return clientStream, err
		}
		// 从池中获取字段切片
		fields := *pool.Get().(*[]slog.Attr)
		// 添加基本日志字段
		fields = append(fields,
			slog.String("system", "grpc.client"),
			slog.String("timestamp", startTime.Format(time.RFC3339)),
			slog.String("latency", time.Since(startTime).String()),
			slogx.Uint("status", status.Code(err)),
			slogx.Error("error", err),
		)
		// 添加服务端地址信息
		if peer, ok := peer.FromContext(ctx); ok {
			fields = append(fields, slog.String("peer", peer.Addr.String()))
		}
		// 添加截止时间信息
		if d, ok := ctx.Deadline(); ok {
			fields = append(fields, slog.String("deadline", d.Format(time.RFC3339)))
		}
		// 记录日志
		logger.LogAttrs(ctx, o.level, method, fields...)
		// 重置切片长度以便复用
		fields = fields[:0]
		// 将切片放回池中
		pool.Put(&fields)
		return clientStream, err
	}
}
