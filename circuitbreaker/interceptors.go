package circuitbreaker

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UnaryClientInterceptor(opts ...Option) grpc.UnaryClientInterceptor {
	breaker := defaultOptions().apply(opts...).init().newSREBreaker()

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

		// 执行实际的gRPC调用
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
