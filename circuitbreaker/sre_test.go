package circuitbreaker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSREBreaker_Allow(t *testing.T) {
	t.Run("initially_allowed", func(t *testing.T) {
		b := defaultOptions().apply(WithK(2.0), WithWindow(time.Second), WithBuckets(10)).init().newSREBreaker()
		assert.True(t, b.Allow())
	})

	t.Run("high_success_rate", func(t *testing.T) {
		b := defaultOptions().apply(WithK(2.0), WithWindow(time.Second), WithBuckets(10)).init().newSREBreaker()
		for i := 0; i < 100; i++ {
			b.MarkSuccess()
		}
		assert.True(t, b.Allow())
	})

	t.Run("all_failures", func(t *testing.T) {
		b := defaultOptions().apply(WithK(2.0), WithWindow(time.Second), WithBuckets(10)).init().newSREBreaker()
		for i := 0; i < 100; i++ {
			b.MarkFailure()
		}
		// With 100 requests and 0 accepts:
		// P = (100 - 2*0) / (100 + 1) = 100/101 ~= 0.99
		// Most requests should be dropped.
		dropped := 0
		for i := 0; i < 100; i++ {
			if !b.Allow() {
				dropped++
			}
		}
		assert.Greater(t, dropped, 80)
	})

	t.Run("k_multiplier", func(t *testing.T) {
		// K=1.1, more aggressive dropping
		b := defaultOptions().apply(WithK(1.1), WithWindow(time.Second), WithBuckets(10)).init().newSREBreaker()
		for i := 0; i < 100; i++ {
			b.MarkSuccess() // 100 accepts
		}
		for i := 0; i < 50; i++ {
			b.MarkFailure() // 150 total requests
		}
		// requests = 150, accepts = 100, K = 1.1
		// P = (150 - 1.1 * 100) / (151) = (150 - 110) / 151 = 40 / 151 ~= 0.26
		// Should drop some requests
		dropped := 0
		for i := 0; i < 1000; i++ {
			if !b.Allow() {
				dropped++
			}
		}
		assert.Greater(t, dropped, 100)
		assert.Less(t, dropped, 400)
	})
}

func TestWindow_Rotate(t *testing.T) {
	w := newWindow(time.Millisecond*100, 10)
	w.Add(10, 5)

	req, acc := w.Summary()
	assert.Equal(t, int64(10), req)
	assert.Equal(t, int64(5), acc)

	// Wait for window to pass
	time.Sleep(time.Millisecond * 150)
	w.Add(1, 1) // Force rotation

	req, acc = w.Summary()
	// Old bucket should be cleared
	assert.Equal(t, int64(1), req)
	assert.Equal(t, int64(1), acc)
}

func TestUnaryClientInterceptor(t *testing.T) {
	t.Run("allow_and_success", func(t *testing.T) {
		interceptor := UnaryClientInterceptor(WithK(2.0))
		invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			return nil
		}

		err := interceptor(context.Background(), "/test/method", nil, nil, nil, invoker)
		assert.NoError(t, err)
	})

	t.Run("allow_and_failure", func(t *testing.T) {
		interceptor := UnaryClientInterceptor(WithK(2.0))
		// Use a code that triggers failure count
		invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			return status.Error(codes.Internal, "internal error")
		}

		err := interceptor(context.Background(), "/test/method", nil, nil, nil, invoker)
		assert.Error(t, err)
		assert.Equal(t, codes.Internal, status.Code(err))
	})

	t.Run("dropped", func(t *testing.T) {
		interceptor := UnaryClientInterceptor(WithK(1.0)) // Aggressive

		// Mock failures to trigger drop
		failInvoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			return status.Error(codes.Internal, "internal error")
		}

		// Fill window with failures
		for i := 0; i < 100; i++ {
			_ = interceptor(context.Background(), "/test/method", nil, nil, nil, failInvoker)
		}

		// Now it should drop
		successInvoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			return nil
		}

		dropped := 0
		for i := 0; i < 100; i++ {
			err := interceptor(context.Background(), "/test/method", nil, nil, nil, successInvoker)
			if status.Code(err) == codes.ResourceExhausted {
				dropped++
			}
		}
		assert.Greater(t, dropped, 50)
	})
}
