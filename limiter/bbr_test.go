package limiter

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/soyacen/grpc-middleware/internal/container"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestBBR_Allow(t *testing.T) {
	t.Run("initially_allowed", func(t *testing.T) {
		l := defaultOptions().init().newLimiter()
		done, err := l.Allow()
		assert.NoError(t, err)
		assert.NotNil(t, done)
		done(DoneInfo{})
	})

	t.Run("limit_exceeded", func(t *testing.T) {
		l := &bbrLimiter{
			conf: &options{
				Window:       time.Second,
				Buckets:      10,
				CPUThreshold: 0.5,
			},
			passStat: container.NewRollingCounter(time.Second, 10, false),
			rtStat:   container.NewRollingCounter(time.Second, 10, true),
			cpu:      func() float64 { return 0.8 },
		}

		for i := 0; i < 100; i++ {
			done, _ := l.Allow()
			if done != nil {
				done(DoneInfo{})
			}
		}

		l.inflight = 1000

		_, err := l.Allow()
		assert.Error(t, err)
		assert.Equal(t, codes.ResourceExhausted, status.Code(err))
	})

	t.Run("cold_start_allows_concurrent_requests", func(t *testing.T) {
		l := &bbrLimiter{
			conf: &options{
				Window:       time.Second,
				Buckets:      10,
				CPUThreshold: 0.5,
			},
			passStat: container.NewRollingCounter(time.Second, 10, false),
			rtStat:   container.NewRollingCounter(time.Second, 10, true),
			cpu:      func() float64 { return 0.9 },
		}

		var wg sync.WaitGroup
		allowed := 0
		var mu sync.Mutex
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				done, err := l.Allow()
				if err == nil && done != nil {
					mu.Lock()
					allowed++
					mu.Unlock()
					time.Sleep(10 * time.Millisecond)
					done(DoneInfo{})
				}
			}()
		}
		wg.Wait()
		assert.Greater(t, allowed, 0)
	})

	t.Run("done_callback_updates_stats", func(t *testing.T) {
		l := &bbrLimiter{
			conf:     defaultOptions().init(),
			passStat: container.NewRollingCounter(time.Second*10, 100, false),
			rtStat:   container.NewRollingCounter(time.Second*10, 100, true),
			cpu:      func() float64 { return 0 },
		}
		done, err := l.Allow()
		assert.NoError(t, err)
		time.Sleep(5 * time.Millisecond)
		done(DoneInfo{Err: nil})

		assert.GreaterOrEqual(t, l.passStat.Max(time.Now()), int64(1))
		assert.GreaterOrEqual(t, l.rtStat.Min(time.Now()), int64(1))
	})

	t.Run("done_callback_skips_stats_on_error", func(t *testing.T) {
		l := &bbrLimiter{
			conf:     defaultOptions().init(),
			passStat: container.NewRollingCounter(time.Second*10, 100, false),
			rtStat:   container.NewRollingCounter(time.Second*10, 100, true),
			cpu:      func() float64 { return 0 },
		}
		initialMax := l.passStat.Max(time.Now())
		done, err := l.Allow()
		assert.NoError(t, err)
		done(DoneInfo{Err: fmt.Errorf("some error")})

		assert.Equal(t, initialMax, l.passStat.Max(time.Now()))
	})

	t.Run("shouldDrop_cpu_below_threshold", func(t *testing.T) {
		l := &bbrLimiter{
			conf: &options{
				CPUThreshold: 0.8,
				Buckets:      10,
			},
			passStat: container.NewRollingCounter(time.Second, 10, false),
			rtStat:   container.NewRollingCounter(time.Second, 10, true),
			cpu:      func() float64 { return 0.1 },
		}
		assert.False(t, l.shouldDrop())
	})

	t.Run("shouldDrop_inflight_one_always_allowed", func(t *testing.T) {
		l := &bbrLimiter{
			conf: &options{
				CPUThreshold: 0.5,
				Buckets:      10,
			},
			passStat: container.NewRollingCounter(time.Second, 10, false),
			rtStat:   container.NewRollingCounter(time.Second, 10, true),
			cpu:      func() float64 { return 1.0 },
			inflight: 1,
		}
		assert.False(t, l.shouldDrop())
	})
}

func TestInterceptors(t *testing.T) {
	t.Run("unary_success", func(t *testing.T) {
		mdw := UnaryServerInterceptor()
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return "ok", nil
		}
		resp, err := mdw(context.Background(), "req", nil, handler)
		assert.NoError(t, err)
		assert.Equal(t, "ok", resp)
	})

	t.Run("unary_skip", func(t *testing.T) {
		skipCalled := false
		mdw := UnaryServerInterceptor(WithSkip(func() bool {
			skipCalled = true
			return true
		}))
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return "skipped", nil
		}
		resp, err := mdw(context.Background(), "req", nil, handler)
		assert.NoError(t, err)
		assert.Equal(t, "skipped", resp)
		assert.True(t, skipCalled, "skip function should be called")
	})

	t.Run("unary_skip_false_goes_to_limiter", func(t *testing.T) {
		mdw := UnaryServerInterceptor(WithSkip(func() bool {
			return false
		}))
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return "ok", nil
		}
		resp, err := mdw(context.Background(), "req", nil, handler)
		assert.NoError(t, err)
		assert.Equal(t, "ok", resp)
	})

	t.Run("unary_panic_recovery_calls_done", func(t *testing.T) {
		l := defaultOptions().init().newLimiter()
		done, _ := l.Allow()
		assert.NotNil(t, done)

		mdw := UnaryServerInterceptor()
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			panic("test panic")
		}

		assert.Panics(t, func() {
			mdw(context.Background(), "req", nil, handler)
		})
	})

	t.Run("stream_success", func(t *testing.T) {
		mdw := StreamServerInterceptor()
		handler := func(srv interface{}, stream grpc.ServerStream) error {
			return nil
		}
		err := mdw(nil, nil, nil, handler)
		assert.NoError(t, err)
	})

	t.Run("stream_skip", func(t *testing.T) {
		skipCalled := false
		mdw := StreamServerInterceptor(WithSkip(func() bool {
			skipCalled = true
			return true
		}))
		handler := func(srv interface{}, stream grpc.ServerStream) error {
			return nil
		}
		err := mdw(nil, nil, nil, handler)
		assert.NoError(t, err)
		assert.True(t, skipCalled)
	})

	t.Run("stream_panic_recovery_calls_done", func(t *testing.T) {
		mdw := StreamServerInterceptor()
		handler := func(srv interface{}, stream grpc.ServerStream) error {
			panic("test panic")
		}
		assert.Panics(t, func() {
			mdw(nil, nil, nil, handler)
		})
	})
}

func TestOptions(t *testing.T) {
	t.Run("default_options", func(t *testing.T) {
		o := defaultOptions()
		assert.Equal(t, time.Second*10, o.Window)
		assert.Equal(t, 100, o.Buckets)
		assert.Equal(t, 0.8, o.CPUThreshold)
		assert.NotNil(t, o.CPU)
		assert.Equal(t, time.Millisecond*500, o.CPUInterval)
	})

	t.Run("with_window", func(t *testing.T) {
		o := defaultOptions().apply(WithWindow(time.Second * 5)).init()
		assert.Equal(t, time.Second*5, o.Window)
	})

	t.Run("with_buckets", func(t *testing.T) {
		o := defaultOptions().apply(WithBuckets(50)).init()
		assert.Equal(t, 50, o.Buckets)
	})

	t.Run("with_cpu_threshold", func(t *testing.T) {
		o := defaultOptions().apply(WithCPUThreshold(0.9)).init()
		assert.Equal(t, 0.9, o.CPUThreshold)
	})

	t.Run("with_cpu", func(t *testing.T) {
		customCPU := func() float64 { return 0.5 }
		o := defaultOptions().apply(WithCPU(customCPU)).init()
		assert.NotNil(t, o.CPU)
		assert.Equal(t, 0.5, o.CPU())
	})

	t.Run("with_cpu_interval", func(t *testing.T) {
		o := defaultOptions().apply(WithCPUInterval(time.Second)).init()
		assert.Equal(t, time.Second, o.CPUInterval)
	})

	t.Run("with_skip", func(t *testing.T) {
		o := defaultOptions().apply(WithSkip(func() bool { return true })).init()
		assert.NotNil(t, o.Skip)
		assert.True(t, o.Skip())
	})

	t.Run("init_fixes_invalid_values", func(t *testing.T) {
		o := &options{
			Window:       0,
			Buckets:      0,
			CPUThreshold: 0,
			CPUInterval:  0,
			CPU:          nil,
		}
		o.init()
		assert.Equal(t, time.Second*10, o.Window)
		assert.Equal(t, 100, o.Buckets)
		assert.Equal(t, 0.8, o.CPUThreshold)
		assert.Equal(t, time.Millisecond*500, o.CPUInterval)
		assert.NotNil(t, o.CPU)
	})
}

func TestLimiterInterface(t *testing.T) {
	t.Run("bbr_limiter_implements_limiter", func(t *testing.T) {
		var _ Limiter = (&bbrLimiter{})
	})
}

func TestMaxInflight(t *testing.T) {
	t.Run("cold_start_returns_buckets", func(t *testing.T) {
		l := &bbrLimiter{
			conf: &options{
				Window:  time.Second,
				Buckets: 10,
			},
			passStat: container.NewRollingCounter(time.Second, 10, false),
			rtStat:   container.NewRollingCounter(time.Second, 10, true),
		}
		assert.Equal(t, float64(10), l.maxInflight())
	})

	t.Run("with_data_returns_calculated_value", func(t *testing.T) {
		l := &bbrLimiter{
			conf: &options{
				Window:  time.Second,
				Buckets: 10,
			},
			passStat: container.NewRollingCounter(time.Second, 10, false),
			rtStat:   container.NewRollingCounter(time.Second, 10, true),
		}
		for i := 0; i < 10; i++ {
			l.passStat.Add(time.Now(), 5)
			l.rtStat.Add(time.Now(), 10)
		}
		result := l.maxInflight()
		assert.Greater(t, result, float64(0))
	})
}

func TestConcurrentLimiter(t *testing.T) {
	t.Run("concurrent_allow_does_not_race", func(t *testing.T) {
		l := &bbrLimiter{
			conf:     defaultOptions().init(),
			passStat: container.NewRollingCounter(time.Second*10, 100, false),
			rtStat:   container.NewRollingCounter(time.Second*10, 100, true),
			cpu:      func() float64 { return 0 },
		}
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				done, err := l.Allow()
				if err == nil && done != nil {
					time.Sleep(time.Millisecond)
					done(DoneInfo{})
				}
			}()
		}
		wg.Wait()
	})
}
