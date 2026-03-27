package retry

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func UnaryClientInterceptor(opts ...Option) grpc.UnaryClientInterceptor {
	o := defaultOptions().apply(opts...)

	return func(
		ctx context.Context,
		method string,
		req interface{},
		reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		grpcOpts ...grpc.CallOption,
	) error {
		var lastErr error

		for attempt := 0; attempt <= o.MaxRetries; attempt++ {
			if attempt > 0 {
				delay := o.BackoffFunc(ctx, uint(attempt))
				if err := sleepWithContext(ctx, delay); err != nil {
					return status.Error(codes.Canceled, "retry: context canceled during backoff")
				}
			}

			callCtx := ctx
			if o.PerCallTimeout > 0 {
				timeoutCtx, cancel := context.WithTimeout(ctx, o.PerCallTimeout)
				lastErr = invoker(timeoutCtx, method, req, reply, cc, grpcOpts...)
				cancel()
			} else {
				lastErr = invoker(callCtx, method, req, reply, cc, grpcOpts...)
			}

			if lastErr == nil {
				return nil
			}

			if !o.RetryableFunc(lastErr) {
				return lastErr
			}

			if ctx.Err() != nil {
				return status.Error(codes.Canceled, "retry: context canceled")
			}
		}

		return lastErr
	}
}

func StreamClientInterceptor(opts ...Option) grpc.StreamClientInterceptor {
	o := defaultOptions().apply(opts...)

	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		grpcOpts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		var lastErr error
		var stream grpc.ClientStream

		for attempt := 0; attempt <= o.MaxRetries; attempt++ {
			if attempt > 0 {
				delay := o.BackoffFunc(ctx, uint(attempt))
				if err := sleepWithContext(ctx, delay); err != nil {
					return nil, status.Error(codes.Canceled, "retry: context canceled during backoff")
				}
			}

			if o.PerCallTimeout > 0 {
				timeoutCtx, cancel := context.WithTimeout(ctx, o.PerCallTimeout)
				stream, lastErr = streamer(timeoutCtx, desc, cc, method, grpcOpts...)
				if lastErr != nil {
					cancel()
				} else {
					stream = &clientStreamWithCancel{ClientStream: stream, cancel: cancel}
				}
			} else {
				stream, lastErr = streamer(ctx, desc, cc, method, grpcOpts...)
			}

			if lastErr == nil {
				return stream, nil
			}

			if !o.RetryableFunc(lastErr) {
				return nil, lastErr
			}

			if ctx.Err() != nil {
				return nil, status.Error(codes.Canceled, "retry: context canceled")
			}
		}

		return nil, lastErr
	}
}

type clientStreamWithCancel struct {
	grpc.ClientStream
	cancel context.CancelFunc
}

func (s *clientStreamWithCancel) RecvMsg(m interface{}) error {
	err := s.ClientStream.RecvMsg(m)
	if err != nil {
		s.cancel()
	}
	return err
}

func (s *clientStreamWithCancel) CloseSend() error {
	s.cancel()
	return s.ClientStream.CloseSend()
}
