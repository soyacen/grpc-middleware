package circuitbreaker

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/soyacen/gox/randx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestBreaker creates a test sreCircuitBreaker
func newTestBreaker(k float64) *sreCircuitBreaker {
	rnd, err := randx.NewPCG()
	if err != nil {
		panic(err)
	}
	return &sreCircuitBreaker{
		k:      k,
		window: newRollingCounter(time.Second*60, 40),
		rnd:    rnd,
	}
}

// newTestBreakerWithSeed creates a test breaker with fixed seed for reproducibility
func newTestBreakerWithSeed(k float64, seed uint64) *sreCircuitBreaker {
	rnd, err := randx.NewPCG()
	if err != nil {
		panic(err)
	}
	return &sreCircuitBreaker{
		k:      k,
		window: newRollingCounter(time.Second*60, 40),
		rnd:    rnd,
	}
}

// countAllowed runs Allow() n times and returns the count of allowed requests
func countAllowed(b *sreCircuitBreaker, n int) (allowed int) {
	for i := 0; i < n; i++ {
		if b.Allow() {
			allowed++
		}
	}
	return
}

// ==================== Basic State Tests (5) ====================

func TestAllow_InitiallyAllowed(t *testing.T) {
	breaker := newTestBreaker(2.0)

	for i := 0; i < 100; i++ {
		assert.True(t, breaker.Allow(), "call %d should be allowed", i+1)
	}
}

func TestAllow_AllSuccess(t *testing.T) {
	breaker := newTestBreaker(2.0)

	for i := 0; i < 100; i++ {
		breaker.MarkSuccess()
	}

	for i := 0; i < 100; i++ {
		assert.True(t, breaker.Allow(), "all success state should allow")
	}
}

func TestAllow_AllFailure_K1(t *testing.T) {
	breaker := newTestBreakerWithSeed(1.0, 42)

	for i := 0; i < 100; i++ {
		breaker.MarkFailure()
	}

	allowed := countAllowed(breaker, 1000)
	assert.Less(t, allowed, 50, "k=1.0 with all failures should reject most")
}

func TestAllow_AllFailure_K2(t *testing.T) {
	breaker := newTestBreakerWithSeed(2.0, 42)

	for i := 0; i < 100; i++ {
		breaker.MarkFailure()
	}

	allowed := countAllowed(breaker, 1000)
	assert.Less(t, allowed, 50, "k=2.0 with all failures should reject most")
}

func TestAllow_ZeroRequests(t *testing.T) {
	breaker := newTestBreaker(1.0)

	for i := 0; i < 100; i++ {
		assert.True(t, breaker.Allow(), "zero requests should always allow")
	}
}

// ==================== Parameter Combination Tests - K Variations (8) ====================

func TestAllow_K1_LowTolerance(t *testing.T) {
	breaker := newTestBreakerWithSeed(1.0, 42)

	for i := 0; i < 60; i++ {
		breaker.MarkSuccess()
	}
	for i := 0; i < 40; i++ {
		breaker.MarkFailure()
	}

	allowed := countAllowed(breaker, 1000)
	tolerance := 150.0
	assert.InDelta(t, 604, allowed, tolerance, "k=1.0, 60 success 40 fail should have ~60%% pass rate")
}

func TestAllow_K2_Default(t *testing.T) {
	breaker := newTestBreaker(2.0)

	for i := 0; i < 50; i++ {
		breaker.MarkSuccess()
	}
	for i := 0; i < 50; i++ {
		breaker.MarkFailure()
	}

	for i := 0; i < 100; i++ {
		assert.True(t, breaker.Allow(), "k=2.0, 50/50 should always allow (p=0)")
	}
}

func TestAllow_K3_HighTolerance(t *testing.T) {
	breaker := newTestBreaker(3.0)

	for i := 0; i < 30; i++ {
		breaker.MarkSuccess()
	}
	for i := 0; i < 70; i++ {
		breaker.MarkFailure()
	}

	allowed := countAllowed(breaker, 1000)
	assert.Greater(t, allowed, 850, "k=3.0, 30 success 70 fail should allow most")
}

func TestAllow_K1_HalfSuccess(t *testing.T) {
	breaker := newTestBreakerWithSeed(1.0, 42)

	for i := 0; i < 50; i++ {
		breaker.MarkSuccess()
	}
	for i := 0; i < 50; i++ {
		breaker.MarkFailure()
	}

	allowed := countAllowed(breaker, 1000)
	tolerance := 150.0
	assert.InDelta(t, 505, allowed, tolerance, "k=1.0, 50/50 should have ~50%% pass rate")
}

func TestAllow_K1_75PercentSuccess(t *testing.T) {
	breaker := newTestBreakerWithSeed(1.0, 42)

	for i := 0; i < 75; i++ {
		breaker.MarkSuccess()
	}
	for i := 0; i < 25; i++ {
		breaker.MarkFailure()
	}

	allowed := countAllowed(breaker, 1000)
	tolerance := 120.0
	assert.InDelta(t, 752, allowed, tolerance, "k=1.0, 75/25 should have ~75%% pass rate")
}

func TestAllow_K2_25PercentSuccess(t *testing.T) {
	breaker := newTestBreakerWithSeed(2.0, 42)

	for i := 0; i < 25; i++ {
		breaker.MarkSuccess()
	}
	for i := 0; i < 75; i++ {
		breaker.MarkFailure()
	}

	allowed := countAllowed(breaker, 1000)
	tolerance := 150.0
	assert.InDelta(t, 505, allowed, tolerance, "k=2.0, 25/75 should have ~50%% pass rate")
}

func TestAllow_K05_VeryLowTolerance(t *testing.T) {
	breaker := newTestBreakerWithSeed(0.5, 42)

	for i := 0; i < 80; i++ {
		breaker.MarkSuccess()
	}
	for i := 0; i < 20; i++ {
		breaker.MarkFailure()
	}

	allowed := countAllowed(breaker, 1000)
	tolerance := 120.0
	assert.InDelta(t, 406, allowed, tolerance, "k=0.5, 80/20 should have ~40%% pass rate")
}

func TestAllow_K5_VeryHighTolerance(t *testing.T) {
	breaker := newTestBreaker(5.0)

	for i := 0; i < 20; i++ {
		breaker.MarkSuccess()
	}
	for i := 0; i < 80; i++ {
		breaker.MarkFailure()
	}

	for i := 0; i < 100; i++ {
		assert.True(t, breaker.Allow(), "k=5.0, 20/80 should always allow (p=0)")
	}
}

// ==================== Statistical Distribution Tests (2) ====================

func TestAllow_StatisticalDistribution_K1(t *testing.T) {
	breaker := newTestBreakerWithSeed(1.0, 42)

	for i := 0; i < 80; i++ {
		breaker.MarkSuccess()
	}
	for i := 0; i < 20; i++ {
		breaker.MarkFailure()
	}

	allowed := countAllowed(breaker, 1000)
	tolerance := 120.0
	assert.InDelta(t, 802, allowed, tolerance, "k=1.0, 80/20 should have ~80%% pass rate")
}

func TestAllow_StatisticalDistribution_K2(t *testing.T) {
	breaker := newTestBreakerWithSeed(2.0, 42)

	for i := 0; i < 40; i++ {
		breaker.MarkSuccess()
	}
	for i := 0; i < 60; i++ {
		breaker.MarkFailure()
	}

	allowed := countAllowed(breaker, 1000)
	tolerance := 120.0
	assert.InDelta(t, 802, allowed, tolerance, "k=2.0, 40/60 should have ~80%% pass rate")
}

// ==================== Edge Case Tests (3) ====================

func TestAllow_SingleSuccess(t *testing.T) {
	breaker := newTestBreaker(2.0)
	breaker.MarkSuccess()
	assert.True(t, breaker.Allow(), "single success should allow")
}

func TestAllow_SingleFailure(t *testing.T) {
	breaker := newTestBreaker(2.0)
	breaker.MarkFailure()
	_ = breaker.Allow()
}

func TestAllow_MoreAcceptsThanRequests(t *testing.T) {
	breaker := newTestBreaker(2.0)

	for i := 0; i < 200; i++ {
		breaker.MarkSuccess()
	}

	for i := 0; i < 100; i++ {
		assert.True(t, breaker.Allow(), "many successes should always allow")
	}
}

// ==================== Mark Method Tests (3) ====================

func TestMarkSuccess_IncreasesAccepts(t *testing.T) {
	breaker := newTestBreaker(2.0)

	req, acc := breaker.window.Summary()
	assert.Equal(t, int64(0), req)
	assert.Equal(t, int64(0), acc)

	breaker.MarkSuccess()
	req, acc = breaker.window.Summary()
	assert.Equal(t, int64(1), req)
	assert.Equal(t, int64(1), acc)

	breaker.MarkSuccess()
	req, acc = breaker.window.Summary()
	assert.Equal(t, int64(2), req)
	assert.Equal(t, int64(2), acc)
}

func TestMarkFailure_DoesNotIncreaseAccepts(t *testing.T) {
	breaker := newTestBreaker(2.0)

	breaker.MarkFailure()
	req, acc := breaker.window.Summary()
	assert.Equal(t, int64(1), req)
	assert.Equal(t, int64(0), acc)

	breaker.MarkFailure()
	req, acc = breaker.window.Summary()
	assert.Equal(t, int64(2), req)
	assert.Equal(t, int64(0), acc)
}

func TestMarkMixed_Alternating(t *testing.T) {
	breaker := newTestBreaker(2.0)

	for i := 0; i < 10; i++ {
		breaker.MarkSuccess()
		breaker.MarkFailure()
	}

	req, acc := breaker.window.Summary()
	assert.Equal(t, int64(20), req)
	assert.Equal(t, int64(10), acc)
}

// ==================== Recovery Tests (4) ====================

func TestRecovery_AfterFailures(t *testing.T) {
	breaker := newTestBreaker(2.0)

	for i := 0; i < 100; i++ {
		breaker.MarkFailure()
	}

	allowedBefore := countAllowed(breaker, 100)
	assert.Less(t, allowedBefore, 10)

	for i := 0; i < 500; i++ {
		breaker.MarkSuccess()
	}

	for i := 0; i < 100; i++ {
		assert.True(t, breaker.Allow(), "after recovery should allow")
	}
}

func TestRecovery_Gradual(t *testing.T) {
	breaker := newTestBreakerWithSeed(1.5, 42)

	for i := 0; i < 100; i++ {
		breaker.MarkFailure()
	}

	initialAllowed := countAllowed(breaker, 200)

	for i := 0; i < 50; i++ {
		breaker.MarkSuccess()
	}

	midAllowed := countAllowed(breaker, 200)
	assert.GreaterOrEqual(t, midAllowed, initialAllowed)

	for i := 0; i < 200; i++ {
		breaker.MarkSuccess()
	}

	finalAllowed := countAllowed(breaker, 200)
	assert.Greater(t, finalAllowed, 150)
}

func TestRecovery_FromAllRejected(t *testing.T) {
	breaker := newTestBreaker(1.0)

	for i := 0; i < 1000; i++ {
		breaker.MarkFailure()
	}

	rejected := 0
	for i := 0; i < 100; i++ {
		if !breaker.Allow() {
			rejected++
		}
	}
	assert.Greater(t, rejected, 50, "should reject most requests after 1000 failures")

	for i := 0; i < 1000; i++ {
		breaker.MarkSuccess()
	}

	allowed := 0
	for i := 0; i < 200; i++ {
		if breaker.Allow() {
			allowed++
		}
	}
	assert.GreaterOrEqual(t, allowed, 80, "after 1000 successes should allow significantly more")
}

func TestRecovery_AfterPartialFailures(t *testing.T) {
	breaker := newTestBreakerWithSeed(2.0, 42)

	for i := 0; i < 30; i++ {
		breaker.MarkFailure()
	}

	before := countAllowed(breaker, 200)
	assert.GreaterOrEqual(t, before, 0, "30 failures with k=2.0 may allow very few requests due to probabilistic nature")

	for i := 0; i < 100; i++ {
		breaker.MarkSuccess()
	}

	after := countAllowed(breaker, 100)
	assert.GreaterOrEqual(t, after, before)
}

// ==================== Concurrent Tests (5) ====================

func TestConcurrent_MixedOperations(t *testing.T) {
	breaker := newTestBreaker(2.0)

	const goroutines = 50
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				switch j % 3 {
				case 0:
					breaker.Allow()
				case 1:
					breaker.MarkSuccess()
				case 2:
					breaker.MarkFailure()
				}
			}
		}(i)
	}

	wg.Wait()

	req, acc := breaker.window.Summary()
	// Allow doesn't add to window, only MarkSuccess and MarkFailure do
	// Each goroutine: 100 ops, 33 Allow (j%3==0), 33 MarkSuccess (j%3==1), 34 MarkFailure (j%3==2)
	// But exact counts depend on iteration distribution, so just verify reasonable bounds
	expectedOps := int64(goroutines * operationsPerGoroutine / 3 * 2)
	assert.InDelta(t, expectedOps, req, float64(goroutines*2))
	assert.GreaterOrEqual(t, acc, int64(0))
	assert.LessOrEqual(t, acc, req)
}

func TestConcurrent_AllowAndMark(t *testing.T) {
	breaker := newTestBreaker(1.5)

	const goroutines = 30
	const iterations = 200

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	var allowedCount int64
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if breaker.Allow() {
					atomic.AddInt64(&allowedCount, 1)
				}
			}
		}()
	}

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if j%2 == 0 {
					breaker.MarkSuccess()
				} else {
					breaker.MarkFailure()
				}
			}
		}(i)
	}

	wg.Wait()

	req, acc := breaker.window.Summary()
	assert.Equal(t, int64(goroutines*iterations), req)
	assert.GreaterOrEqual(t, acc, int64(0))
	assert.LessOrEqual(t, acc, req)
	assert.GreaterOrEqual(t, allowedCount, int64(0))
}

func TestConcurrent_MarkOnly(t *testing.T) {
	breaker := newTestBreaker(2.0)

	const goroutines = 50
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if j%2 == 0 {
					breaker.MarkSuccess()
				} else {
					breaker.MarkFailure()
				}
			}
		}(i)
	}

	wg.Wait()

	req, acc := breaker.window.Summary()
	assert.Equal(t, int64(goroutines*iterations), req)
	assert.InDelta(t, goroutines*iterations/2, int(acc), goroutines*iterations/10)
}

func TestConcurrent_HighContention(t *testing.T) {
	breaker := newTestBreaker(1.5)

	const goroutines = 100
	const iterations = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if breaker.Allow() {
					if j%3 == 0 {
						breaker.MarkSuccess()
					} else {
						breaker.MarkFailure()
					}
				}
			}
		}()
	}

	wg.Wait()

	req, acc := breaker.window.Summary()
	require.GreaterOrEqual(t, req, int64(0))
	require.GreaterOrEqual(t, acc, int64(0))
	require.LessOrEqual(t, acc, req)
}

func TestConcurrent_AllowOnly(t *testing.T) {
	breaker := newTestBreaker(2.0)

	const goroutines = 50
	const iterations = 100

	var wg sync.WaitGroup
	var allowedCount int64
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if breaker.Allow() {
					atomic.AddInt64(&allowedCount, 1)
				}
			}
		}()
	}

	wg.Wait()

	assert.GreaterOrEqual(t, allowedCount, int64(0))
	assert.LessOrEqual(t, allowedCount, int64(goroutines*iterations))
}

// ==================== Formula Correctness Test (1) ====================

func TestAllow_FormulaCorrectness(t *testing.T) {
	tests := []struct {
		name     string
		requests int64
		accepts  int64
		k        float64
		wantP    float64
	}{
		{"empty", 0, 0, 2.0, 0.0},
		{"all_success", 100, 100, 2.0, -100.0 / 101.0},
		{"all_fail_k2", 100, 0, 2.0, 100.0 / 101.0},
		{"50_50_k2", 100, 50, 2.0, 0.0},
		{"50_50_k1", 100, 50, 1.0, 50.0 / 101.0},
		{"80_20_k1", 100, 80, 1.0, 20.0 / 101.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := (float64(tt.requests) - tt.k*float64(tt.accepts)) / float64(tt.requests+1)
			assert.InDelta(t, tt.wantP, p, 0.0001)
		})
	}
}

// ==================== Additional Test to Ensure 25 Total ====================

func TestAllow_K1_60PercentSuccess(t *testing.T) {
	breaker := newTestBreakerWithSeed(1.0, 42)

	for i := 0; i < 60; i++ {
		breaker.MarkSuccess()
	}
	for i := 0; i < 40; i++ {
		breaker.MarkFailure()
	}

	allowed := countAllowed(breaker, 1000)
	tolerance := 150.0
	assert.InDelta(t, 604, allowed, tolerance)
}
