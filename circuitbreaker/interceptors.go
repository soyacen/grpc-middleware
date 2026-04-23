package circuitbreaker

import (
	"context"
	"io"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// wrappedClientStream 包装 grpc.ClientStream 以跟踪流错误
type wrappedClientStream struct {
	grpc.ClientStream
	breaker  CircuitBreaker
	markOnce sync.Once
}

// RecvMsg 接收消息并标记熔断状态
func (w *wrappedClientStream) RecvMsg(m interface{}) error {
	err := w.ClientStream.RecvMsg(m)

	if err == nil {
		return nil
	}

	w.markOnce.Do(func() {
		if err == io.EOF {
			// 流正常结束
			w.breaker.MarkSuccess()
			return
		}

		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.DeadlineExceeded,
				codes.Internal,
				codes.Unavailable,
				codes.ResourceExhausted:
				w.breaker.MarkFailure()
			default:
				w.breaker.MarkSuccess()
			}
		} else {
			w.breaker.MarkFailure()
		}
	})

	return err
}

// StreamClientInterceptor 创建流式调用的客户端熔断拦截器
func StreamClientInterceptor(opts ...Option) grpc.StreamClientInterceptor {
	breaker := defaultOptions().apply(opts...).init().newCircuitBreaker()

	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		if !breaker.Allow() {
			return nil, ErrCircuitBreakerOpen
		}

		stream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			st, ok := status.FromError(err)
			if ok {
				switch st.Code() {
				case codes.DeadlineExceeded,
					codes.Internal,
					codes.Unavailable,
					codes.ResourceExhausted:
					breaker.MarkFailure()
				default:
					breaker.MarkSuccess()
				}
			} else {
				breaker.MarkFailure()
			}
			return nil, err
		}

		return &wrappedClientStream{
			ClientStream: stream,
			breaker:      breaker,
		}, nil
	}
}

// UnaryClientInterceptor 创建一元调用的客户端熔断拦截器
func UnaryClientInterceptor(opts ...Option) grpc.UnaryClientInterceptor {
	breaker := defaultOptions().apply(opts...).init().newCircuitBreaker()
	return func(
		ctx context.Context,
		method string,
		req interface{},
		reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		grpcOpts ...grpc.CallOption,
	) error {
		if !breaker.Allow() {
			return ErrCircuitBreakerOpen
		}

		err := invoker(ctx, method, req, reply, cc, grpcOpts...)

		if err == nil {
			breaker.MarkSuccess()
			return nil
		}

		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.DeadlineExceeded,
				codes.Internal,
				codes.Unavailable,
				codes.ResourceExhausted:
				breaker.MarkFailure()
			default:
				breaker.MarkSuccess()
			}
		} else {
			breaker.MarkFailure()
		}

		return err
	}
}
