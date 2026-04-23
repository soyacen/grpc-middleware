package ratelimiter

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
)

// UnaryServerInterceptor 创建一元调用的服务端限流拦截器
func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	o := defaultOptions().apply(opts...).init()
	limiter := o.newRateLimiter()

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if o.Skip != nil && o.Skip(ctx, info.FullMethod) {
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

// StreamServerInterceptor 创建流式调用的服务端限流拦截器
func StreamServerInterceptor(opts ...Option) grpc.StreamServerInterceptor {
	o := defaultOptions().apply(opts...).init()
	limiter := o.newRateLimiter()

	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if o.Skip != nil && o.Skip(stream.Context(), info.FullMethod) {
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
