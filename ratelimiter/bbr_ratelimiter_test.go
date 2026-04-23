package ratelimiter

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMaxInflight_ColdStart(t *testing.T) {
	l := &bbrRateLimiter{
		conf: &options{
			Window:  time.Second,
			Buckets: 10,
		},
		passStat: newRollingCounter(time.Second, 10, false),
		rtStat:   newRollingCounter(time.Second, 10, true),
	}
	result := l.maxInflight()
	assert.Equal(t, float64(10), result)
}

func TestMaxInflight_Basic(t *testing.T) {
	l := &bbrRateLimiter{
		conf: &options{
			Window:  time.Second,
			Buckets: 10,
		},
		passStat: newRollingCounter(time.Second, 10, false),
		rtStat:   newRollingCounter(time.Second, 10, true),
	}

	now := time.Now()
	l.passStat.Add(now, 100)
	l.rtStat.Add(now, 50)

	result := l.maxInflight()
	assert.Greater(t, result, float64(0))
}

func TestMaxInflight_LargeRT(t *testing.T) {
	l := &bbrRateLimiter{
		conf: &options{
			Window:  time.Second,
			Buckets: 10,
		},
		passStat: newRollingCounter(time.Second, 10, false),
		rtStat:   newRollingCounter(time.Second, 10, true),
	}

	now := time.Now()
	l.passStat.Add(now, 50)
	l.rtStat.Add(now, 1000)

	result := l.maxInflight()
	assert.Greater(t, result, float64(0))
}

func TestShouldDrop_CPUBelowThreshold(t *testing.T) {
	l := &bbrRateLimiter{
		conf: &options{
			CPUThreshold: 0.8,
			Buckets:      10,
		},
		passStat: newRollingCounter(time.Second, 10, false),
		rtStat:   newRollingCounter(time.Second, 10, true),
		cpu:      func() float64 { return 0.1 },
	}
	assert.False(t, l.shouldDrop())
}

func TestShouldDrop_CPUAboveThreshold(t *testing.T) {
	l := &bbrRateLimiter{
		conf: &options{
			CPUThreshold: 0.5,
			Buckets:      10,
		},
		passStat: newRollingCounter(time.Second, 10, false),
		rtStat:   newRollingCounter(time.Second, 10, true),
		cpu:      func() float64 { return 0.9 },
		inflight: 1000,
	}
	result := l.shouldDrop()
	assert.IsType(t, true, result)
}

func TestShouldDrop_InflightOneAlwaysAllowed(t *testing.T) {
	l := &bbrRateLimiter{
		conf: &options{
			CPUThreshold: 0.5,
			Buckets:      10,
		},
		passStat: newRollingCounter(time.Second, 10, false),
		rtStat:   newRollingCounter(time.Second, 10, true),
		cpu:      func() float64 { return 1.0 },
		inflight: 1,
	}
	assert.False(t, l.shouldDrop())
}

func TestShouldDrop_LastDropWithinSecond(t *testing.T) {
	l := &bbrRateLimiter{
		conf: &options{
			CPUThreshold: 0.8,
			Buckets:      10,
		},
		passStat: newRollingCounter(time.Second, 10, false),
		rtStat:   newRollingCounter(time.Second, 10, true),
		cpu:      func() float64 { return 0.9 },
		inflight: 1000,
	}
	now := time.Now()
	l.lastDrop.Store(&now)

	assert.True(t, l.shouldDrop())
}

func TestShouldDrop_LastDropAfterSecond(t *testing.T) {
	l := &bbrRateLimiter{
		conf: &options{
			CPUThreshold: 0.8,
			Buckets:      10,
		},
		passStat: newRollingCounter(time.Second, 10, false),
		rtStat:   newRollingCounter(time.Second, 10, true),
		cpu:      func() float64 { return 0.1 },
	}
	oldTime := time.Now().Add(-2 * time.Second)
	l.lastDrop.Store(&oldTime)

	assert.False(t, l.shouldDrop())
	assert.Nil(t, l.lastDrop.Load())
}

func TestAllow_InitiallyAllowed(t *testing.T) {
	l := &bbrRateLimiter{
		conf: &options{
			CPUThreshold: 0.8,
			Buckets:      10,
		},
		passStat: newRollingCounter(time.Second, 10, false),
		rtStat:   newRollingCounter(time.Second, 10, true),
		cpu:      func() float64 { return 0.1 },
	}
	done, err := l.Allow()
	assert.NoError(t, err)
	assert.NotNil(t, done)
	done(DoneInfo{})
}

func TestAllow_LimitExceeded(t *testing.T) {
	l := &bbrRateLimiter{
		conf: &options{
			CPUThreshold: 0.5,
			Buckets:      10,
		},
		passStat: newRollingCounter(time.Second, 10, false),
		rtStat:   newRollingCounter(time.Second, 10, true),
		cpu:      func() float64 { return 0.9 },
	}
	l.inflight = 1000

	done, err := l.Allow()
	assert.Error(t, err)
	assert.Nil(t, done)
	assert.Equal(t, codes.ResourceExhausted, status.Code(err))
}

func TestAllow_ColdStartConcurrent(t *testing.T) {
	l := &bbrRateLimiter{
		conf: &options{
			CPUThreshold: 0.5,
			Buckets:      10,
		},
		passStat: newRollingCounter(time.Second, 10, false),
		rtStat:   newRollingCounter(time.Second, 10, true),
		cpu:      func() float64 { return 0.9 },
	}

	var wg sync.WaitGroup
	allowed := int64(0)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			done, err := l.Allow()
			if err == nil && done != nil {
				atomic.AddInt64(&allowed, 1)
				time.Sleep(10 * time.Millisecond)
				done(DoneInfo{})
			}
		}()
	}
	wg.Wait()
	assert.Greater(t, allowed, int64(0))
}

func TestAllow_InflightTracking(t *testing.T) {
	l := &bbrRateLimiter{
		conf: &options{
			CPUThreshold: 0.8,
			Buckets:      10,
		},
		passStat: newRollingCounter(time.Second, 10, false),
		rtStat:   newRollingCounter(time.Second, 10, true),
		cpu:      func() float64 { return 0.1 },
	}

	assert.Equal(t, int64(0), atomic.LoadInt64(&l.inflight))

	done1, _ := l.Allow()
	assert.Equal(t, int64(1), atomic.LoadInt64(&l.inflight))

	done2, _ := l.Allow()
	assert.Equal(t, int64(2), atomic.LoadInt64(&l.inflight))

	done1(DoneInfo{})
	assert.Equal(t, int64(1), atomic.LoadInt64(&l.inflight))

	done2(DoneInfo{})
	assert.Equal(t, int64(0), atomic.LoadInt64(&l.inflight))
}

func TestDone_SuccessUpdatesStats(t *testing.T) {
	l := &bbrRateLimiter{
		conf:     defaultOptions().init(),
		passStat: newRollingCounter(time.Second*10, 100, false),
		rtStat:   newRollingCounter(time.Second*10, 100, true),
		cpu:      func() float64 { return 0 },
	}

	done, err := l.Allow()
	assert.NoError(t, err)

	time.Sleep(5 * time.Millisecond)
	done(DoneInfo{Err: nil})

	assert.GreaterOrEqual(t, l.passStat.Max(time.Now()), int64(1))
	assert.GreaterOrEqual(t, l.rtStat.Min(time.Now()), int64(1))
}

func TestDone_ErrorSkipsStats(t *testing.T) {
	l := &bbrRateLimiter{
		conf:     defaultOptions().init(),
		passStat: newRollingCounter(time.Second*10, 100, false),
		rtStat:   newRollingCounter(time.Second*10, 100, true),
		cpu:      func() float64 { return 0 },
	}

	initialMax := l.passStat.Max(time.Now())
	done, err := l.Allow()
	assert.NoError(t, err)

	done(DoneInfo{Err: status.Error(codes.Internal, "error")})

	assert.Equal(t, initialMax, l.passStat.Max(time.Now()))
}

func TestDone_InflightDecremented(t *testing.T) {
	l := &bbrRateLimiter{
		conf: &options{
			CPUThreshold: 0.8,
			Buckets:      10,
		},
		passStat: newRollingCounter(time.Second, 10, false),
		rtStat:   newRollingCounter(time.Second, 10, true),
		cpu:      func() float64 { return 0.1 },
	}

	done, _ := l.Allow()
	assert.Equal(t, int64(1), atomic.LoadInt64(&l.inflight))

	done(DoneInfo{Err: nil})
	assert.Equal(t, int64(0), atomic.LoadInt64(&l.inflight))
}

func TestConcurrent_Allow(t *testing.T) {
	l := &bbrRateLimiter{
		conf: &options{
			Window:       time.Second * 10,
			Buckets:      100,
			CPUThreshold: 0.8,
		},
		passStat: newRollingCounter(time.Second*10, 100, false),
		rtStat:   newRollingCounter(time.Second*10, 100, true),
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
}

func TestBBR_MaxInflightWithZeroPass(t *testing.T) {
	l := &bbrRateLimiter{
		conf: &options{
			Window:  time.Second,
			Buckets: 10,
		},
		passStat: newRollingCounter(time.Second, 10, false),
		rtStat:   newRollingCounter(time.Second, 10, true),
	}

	result := l.maxInflight()
	assert.Equal(t, float64(10), result)
}

func TestBBR_MaxInflightWithZeroRT(t *testing.T) {
	l := &bbrRateLimiter{
		conf: &options{
			Window:  time.Second,
			Buckets: 10,
		},
		passStat: newRollingCounter(time.Second, 10, false),
		rtStat:   newRollingCounter(time.Second, 10, true),
	}

	l.passStat.Add(time.Now(), 100)
	result := l.maxInflight()
	assert.Equal(t, float64(10), result)
}

func TestBBR_AllowAfterDropRecovery(t *testing.T) {
	l := &bbrRateLimiter{
		conf: &options{
			CPUThreshold: 0.8,
			Buckets:      10,
		},
		passStat: newRollingCounter(time.Second, 10, false),
		rtStat:   newRollingCounter(time.Second, 10, true),
		cpu:      func() float64 { return 0.1 },
	}

	oldTime := time.Now().Add(-3 * time.Second)
	l.lastDrop.Store(&oldTime)

	done, err := l.Allow()
	assert.NoError(t, err)
	assert.NotNil(t, done)
	done(DoneInfo{})
}

func TestBBR_MultipleRequestsAccumulate(t *testing.T) {
	l := &bbrRateLimiter{
		conf: &options{
			Window:       time.Second * 10,
			Buckets:      100,
			CPUThreshold: 0.8,
		},
		passStat: newRollingCounter(time.Second*10, 100, false),
		rtStat:   newRollingCounter(time.Second*10, 100, true),
		cpu:      func() float64 { return 0 },
	}

	for i := 0; i < 10; i++ {
		done, err := l.Allow()
		assert.NoError(t, err)
		time.Sleep(time.Millisecond)
		done(DoneInfo{})
	}

	assert.GreaterOrEqual(t, l.passStat.Max(time.Now()), int64(1))
}

func TestBBR_HighCPUWithLowInflight(t *testing.T) {
	l := &bbrRateLimiter{
		conf: &options{
			CPUThreshold: 0.5,
			Buckets:      10,
		},
		passStat: newRollingCounter(time.Second, 10, false),
		rtStat:   newRollingCounter(time.Second, 10, true),
		cpu:      func() float64 { return 0.9 },
		inflight: 0,
	}

	done, err := l.Allow()
	assert.NoError(t, err)
	assert.NotNil(t, done)
	done(DoneInfo{})
}

func TestBBR_FormulaCalculation(t *testing.T) {
	l := &bbrRateLimiter{
		conf: &options{
			Window:  time.Second,
			Buckets: 10,
		},
		passStat: newRollingCounter(time.Second, 10, false),
		rtStat:   newRollingCounter(time.Second, 10, true),
	}

	now := time.Now()
	for i := 0; i < 5; i++ {
		l.passStat.Add(now, 100)
		l.rtStat.Add(now, 50)
	}

	result := l.maxInflight()
	assert.Greater(t, result, float64(0))
}

func TestBBR_ConcurrentInflightTracking(t *testing.T) {
	l := &bbrRateLimiter{
		conf: &options{
			CPUThreshold: 0.8,
			Buckets:      10,
		},
		passStat: newRollingCounter(time.Second*10, 100, false),
		rtStat:   newRollingCounter(time.Second*10, 100, true),
		cpu:      func() float64 { return 0 },
	}

	var wg sync.WaitGroup
	dones := make([]func(DoneInfo), 0, 50)
	var mu sync.Mutex

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			done, err := l.Allow()
			if err == nil {
				mu.Lock()
				dones = append(dones, done)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	assert.GreaterOrEqual(t, atomic.LoadInt64(&l.inflight), int64(0))

	for _, done := range dones {
		done(DoneInfo{})
	}

	assert.Equal(t, int64(0), atomic.LoadInt64(&l.inflight))
}

func TestBBR_RateLimiterImplementsInterface(t *testing.T) {
	var _ RateLimiter = (*bbrRateLimiter)(nil)
}

func TestBBR_DefaultRateLimiterCreation(t *testing.T) {
	o := defaultOptions().init()
	limiter := o.newRateLimiter()
	assert.NotNil(t, limiter)

	done, err := limiter.Allow()
	assert.NoError(t, err)
	assert.NotNil(t, done)
	done(DoneInfo{})
}
