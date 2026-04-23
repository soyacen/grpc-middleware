package circuitbreaker

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAddAndSummary(t *testing.T) {
	tests := []struct {
		name         string
		window       time.Duration
		buckets      int
		requests     int64
		accepts      int64
		wantRequests int64
		wantAccepts  int64
	}{
		{
			name:         "single_add",
			window:       time.Second,
			buckets:      10,
			requests:     10,
			accepts:      8,
			wantRequests: 10,
			wantAccepts:  8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newRollingCounter(tt.window, tt.buckets)
			w.Add(tt.requests, tt.accepts)
			gotReq, gotAcc := w.Summary()
			assert.Equal(t, tt.wantRequests, gotReq)
			assert.Equal(t, tt.wantAccepts, gotAcc)
		})
	}
}

func TestAdd_Multiple(t *testing.T) {
	tests := []struct {
		name         string
		adds         []struct{ req, acc int64 }
		wantRequests int64
		wantAccepts  int64
	}{
		{
			name: "three_adds",
			adds: []struct{ req, acc int64 }{
				{5, 5},
				{3, 2},
				{2, 1},
			},
			wantRequests: 10,
			wantAccepts:  8,
		},
		{
			name: "ten_adds",
			adds: []struct{ req, acc int64 }{
				{1, 1}, {2, 2}, {3, 3}, {4, 4}, {5, 5},
				{6, 6}, {7, 7}, {8, 8}, {9, 9}, {10, 10},
			},
			wantRequests: 55,
			wantAccepts:  55,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newRollingCounter(time.Second, 10)
			for _, add := range tt.adds {
				w.Add(add.req, add.acc)
			}
			gotReq, gotAcc := w.Summary()
			assert.Equal(t, tt.wantRequests, gotReq)
			assert.Equal(t, tt.wantAccepts, gotAcc)
		})
	}
}

func TestSummary_Empty(t *testing.T) {
	tests := []struct {
		name         string
		wantRequests int64
		wantAccepts  int64
	}{
		{
			name:         "empty_counter_returns_zero",
			wantRequests: 0,
			wantAccepts:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newRollingCounter(time.Second, 10)
			gotReq, gotAcc := w.Summary()
			assert.Equal(t, tt.wantRequests, gotReq)
			assert.Equal(t, tt.wantAccepts, gotAcc)
		})
	}
}

func TestRotate_ExactBucketDuration(t *testing.T) {
	tests := []struct {
		name         string
		window       time.Duration
		buckets      int
		sleepTime    time.Duration
		wantRequests int64
		wantAccepts  int64
	}{
		{
			name:         "rotate_after_full_window_clears_old_data",
			window:       time.Millisecond * 100,
			buckets:      10,
			sleepTime:    time.Millisecond * 120,
			wantRequests: 1,
			wantAccepts:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newRollingCounter(tt.window, tt.buckets)
			w.Add(10, 5)
			time.Sleep(tt.sleepTime)
			w.Add(1, 1)
			gotReq, gotAcc := w.Summary()
			assert.Equal(t, tt.wantRequests, gotReq)
			assert.Equal(t, tt.wantAccepts, gotAcc)
		})
	}
}

func TestRotate_MultipleBuckets(t *testing.T) {
	tests := []struct {
		name         string
		window       time.Duration
		buckets      int
		sleepTime    time.Duration
		addReq       int64
		addAcc       int64
		wantRequests int64
		wantAccepts  int64
	}{
		{
			name:         "rotate_multiple_buckets_partial_window",
			window:       time.Millisecond * 100,
			buckets:      10,
			sleepTime:    time.Millisecond * 35,
			addReq:       5,
			addAcc:       5,
			wantRequests: 105,
			wantAccepts:  55,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newRollingCounter(tt.window, tt.buckets)
			w.Add(100, 50)
			time.Sleep(tt.sleepTime)
			w.Add(tt.addReq, tt.addAcc)
			gotReq, gotAcc := w.Summary()
			assert.Equal(t, tt.wantRequests, gotReq)
			assert.Equal(t, tt.wantAccepts, gotAcc)
		})
	}
}

func TestRotate_AllBuckets(t *testing.T) {
	tests := []struct {
		name         string
		window       time.Duration
		buckets      int
		sleepTime    time.Duration
		wantRequests int64
		wantAccepts  int64
	}{
		{
			name:         "rotate_all_buckets_clears_all",
			window:       time.Millisecond * 100,
			buckets:      5,
			sleepTime:    time.Millisecond * 110,
			wantRequests: 1,
			wantAccepts:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newRollingCounter(tt.window, tt.buckets)
			w.Add(100, 50)
			time.Sleep(tt.sleepTime)
			w.Add(1, 1)
			gotReq, gotAcc := w.Summary()
			assert.Equal(t, tt.wantRequests, gotReq)
			assert.Equal(t, tt.wantAccepts, gotAcc)
		})
	}
}

func TestRotate_LongElapsed(t *testing.T) {
	tests := []struct {
		name         string
		window       time.Duration
		buckets      int
		sleepTime    time.Duration
		wantRequests int64
		wantAccepts  int64
	}{
		{
			name:         "very_long_elapsed_resets_all",
			window:       time.Millisecond * 100,
			buckets:      10,
			sleepTime:    time.Millisecond * 200,
			wantRequests: 5,
			wantAccepts:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newRollingCounter(tt.window, tt.buckets)
			w.Add(1000, 500)
			time.Sleep(tt.sleepTime)
			w.Add(5, 5)
			gotReq, gotAcc := w.Summary()
			assert.Equal(t, tt.wantRequests, gotReq)
			assert.Equal(t, tt.wantAccepts, gotAcc)
		})
	}
}

func TestAdd_Zero(t *testing.T) {
	tests := []struct {
		name         string
		requests     int64
		accepts      int64
		wantRequests int64
		wantAccepts  int64
	}{
		{
			name:         "add_zero_values",
			requests:     0,
			accepts:      0,
			wantRequests: 0,
			wantAccepts:  0,
		},
		{
			name:         "add_zero_requests_positive_accepts",
			requests:     0,
			accepts:      5,
			wantRequests: 0,
			wantAccepts:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newRollingCounter(time.Second, 10)
			w.Add(tt.requests, tt.accepts)
			gotReq, gotAcc := w.Summary()
			assert.Equal(t, tt.wantRequests, gotReq)
			assert.Equal(t, tt.wantAccepts, gotAcc)
		})
	}
}

func TestAdd_Negative(t *testing.T) {
	tests := []struct {
		name         string
		requests     int64
		accepts      int64
		wantRequests int64
		wantAccepts  int64
	}{
		{
			name:         "add_negative_values",
			requests:     -5,
			accepts:      -3,
			wantRequests: -5,
			wantAccepts:  -3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newRollingCounter(time.Second, 10)
			w.Add(tt.requests, tt.accepts)
			gotReq, gotAcc := w.Summary()
			assert.Equal(t, tt.wantRequests, gotReq)
			assert.Equal(t, tt.wantAccepts, gotAcc)
		})
	}
}

func TestConcurrent_Add(t *testing.T) {
	tests := []struct {
		name         string
		goroutines   int
		wantRequests int64
		wantAccepts  int64
	}{
		{
			name:         "100_concurrent_adds",
			goroutines:   100,
			wantRequests: 100,
			wantAccepts:  100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newRollingCounter(time.Second, 10)
			var wg sync.WaitGroup

			for i := 0; i < tt.goroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					w.Add(1, 1)
				}()
			}

			wg.Wait()
			gotReq, gotAcc := w.Summary()
			assert.Equal(t, tt.wantRequests, gotReq)
			assert.Equal(t, tt.wantAccepts, gotAcc)
		})
	}
}

func TestConcurrent_AddAndSummary(t *testing.T) {
	tests := []struct {
		name       string
		addWorkers int
		sumWorkers int
	}{
		{
			name:       "concurrent_add_and_summary",
			addWorkers: 50,
			sumWorkers: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newRollingCounter(time.Second, 10)
			var wg sync.WaitGroup
			done := make(chan struct{})

			for i := 0; i < tt.addWorkers; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < 10; j++ {
						w.Add(1, 1)
					}
				}()
			}

			for i := 0; i < tt.sumWorkers; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < 10; j++ {
						w.Summary()
					}
				}()
			}

			go func() {
				wg.Wait()
				close(done)
			}()

			select {
			case <-done:
				gotReq, gotAcc := w.Summary()
				assert.Equal(t, int64(500), gotReq)
				assert.Equal(t, int64(500), gotAcc)
			case <-time.After(time.Second * 2):
				t.Fatal("timeout waiting for concurrent operations")
			}
		})
	}
}

func TestConcurrent_Rotate(t *testing.T) {
	tests := []struct {
		name     string
		workers  int
		duration time.Duration
	}{
		{
			name:     "concurrent_rotate_operations",
			workers:  10,
			duration: time.Millisecond * 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newRollingCounter(tt.duration, 5)
			var wg sync.WaitGroup

			for i := 0; i < tt.workers; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					w.Add(1, 1)
					time.Sleep(tt.duration / 2)
					w.Add(1, 1)
				}()
			}

			wg.Wait()
			gotReq, _ := w.Summary()
			assert.Equal(t, int64(tt.workers*2), gotReq)
		})
	}
}

func TestTimePrecision_BucketAlignment(t *testing.T) {
	tests := []struct {
		name    string
		window  time.Duration
		buckets int
	}{
		{
			name:    "bucket_size_calculated_correctly",
			window:  time.Millisecond * 100,
			buckets: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newRollingCounter(tt.window, tt.buckets)
			expectedBucketSize := tt.window / time.Duration(tt.buckets)
			assert.Equal(t, expectedBucketSize, w.bucketSize)
		})
	}
}

func TestRotate_NoRotationWithinBucket(t *testing.T) {
	tests := []struct {
		name         string
		window       time.Duration
		buckets      int
		adds         int
		wantRequests int64
	}{
		{
			name:         "no_rotation_within_bucket_duration",
			window:       time.Second,
			buckets:      10,
			adds:         3,
			wantRequests: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newRollingCounter(tt.window, tt.buckets)
			for i := 0; i < tt.adds; i++ {
				w.Add(10, 5)
				if i < tt.adds-1 {
					gotReq, _ := w.Summary()
					assert.Equal(t, int64(10*(i+1)), gotReq)
				}
			}
			gotReq, gotAcc := w.Summary()
			assert.Equal(t, tt.wantRequests, gotReq)
			assert.Equal(t, int64(15), gotAcc)
		})
	}
}

func TestRotate_WithinWindow(t *testing.T) {
	tests := []struct {
		name         string
		window       time.Duration
		buckets      int
		wantRequests int64
	}{
		{
			name:         "rotation_within_window_keeps_partial_data",
			window:       time.Millisecond * 100,
			buckets:      10,
			wantRequests: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newRollingCounter(tt.window, tt.buckets)
			w.Add(5, 3)
			time.Sleep(time.Millisecond * 12)
			w.Add(3, 2)
			gotReq, gotAcc := w.Summary()
			assert.Equal(t, tt.wantRequests, gotReq)
			assert.Equal(t, int64(5), gotAcc)
		})
	}
}

func TestAdd_ConsecutiveSameBucket(t *testing.T) {
	tests := []struct {
		name         string
		adds         []struct{ req, acc int64 }
		wantRequests int64
		wantAccepts  int64
	}{
		{
			name: "consecutive_adds_same_bucket",
			adds: []struct{ req, acc int64 }{
				{1, 1}, {2, 2}, {3, 3}, {4, 4},
			},
			wantRequests: 10,
			wantAccepts:  10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newRollingCounter(time.Second, 10)
			for _, add := range tt.adds {
				w.Add(add.req, add.acc)
			}
			gotReq, gotAcc := w.Summary()
			assert.Equal(t, tt.wantRequests, gotReq)
			assert.Equal(t, tt.wantAccepts, gotAcc)
		})
	}
}

func TestSummary_AfterRotation(t *testing.T) {
	tests := []struct {
		name         string
		window       time.Duration
		buckets      int
		sleepTime    time.Duration
		wantRequests int64
	}{
		{
			name:         "summary_returns_correct_after_rotation",
			window:       time.Millisecond * 100,
			buckets:      5,
			sleepTime:    time.Millisecond * 50,
			wantRequests: 17,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newRollingCounter(tt.window, tt.buckets)
			w.Add(10, 5)
			time.Sleep(tt.sleepTime)
			w.Add(7, 6)
			gotReq, gotAcc := w.Summary()
			assert.Equal(t, tt.wantRequests, gotReq)
			assert.Equal(t, int64(11), gotAcc)
		})
	}
}

func TestNewRollingCounter(t *testing.T) {
	tests := []struct {
		name        string
		window      time.Duration
		buckets     int
		wantBuckets int
	}{
		{
			name:        "create_with_valid_params",
			window:      time.Second,
			buckets:     10,
			wantBuckets: 10,
		},
		{
			name:        "create_with_single_bucket",
			window:      time.Second,
			buckets:     1,
			wantBuckets: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newRollingCounter(tt.window, tt.buckets)
			assert.NotNil(t, w)
			assert.Len(t, w.buckets, tt.wantBuckets)
			assert.Equal(t, tt.window, w.windowSize)
		})
	}
}
