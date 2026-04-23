package ratelimiter

import (
	"math"
	"sync"
	"testing"
	"time"
)

func TestNewRollingCounter_Sum(t *testing.T) {
	tests := []struct {
		name    string
		window  time.Duration
		buckets int
		wantLen int
	}{
		{"10_buckets", time.Second, 10, 10},
		{"100_buckets", time.Second * 10, 100, 100},
		{"5_buckets", time.Millisecond * 50, 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newRollingCounter(tt.window, tt.buckets, false)
			if len(c.buckets) != tt.wantLen {
				t.Errorf("buckets len = %d, want %d", len(c.buckets), tt.wantLen)
			}
			for i, v := range c.buckets {
				if v != 0 {
					t.Errorf("bucket[%d] = %d, want 0", i, v)
				}
			}
		})
	}
}

func TestNewRollingCounter_Min(t *testing.T) {
	tests := []struct {
		name    string
		window  time.Duration
		buckets int
		wantLen int
	}{
		{"10_buckets", time.Second, 10, 10},
		{"100_buckets", time.Second * 10, 100, 100},
		{"5_buckets", time.Millisecond * 50, 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newRollingCounter(tt.window, tt.buckets, true)
			if len(c.buckets) != tt.wantLen {
				t.Errorf("buckets len = %d, want %d", len(c.buckets), tt.wantLen)
			}
			for i, v := range c.buckets {
				if v != math.MaxInt64 {
					t.Errorf("bucket[%d] = %d, want MaxInt64", i, v)
				}
			}
		})
	}
}

func TestAdd(t *testing.T) {
	tests := []struct {
		name   string
		isMin  bool
		adds   []int64
		want   int64
		method string
	}{
		{"sum_single", false, []int64{10}, 10, "Max"},
		{"sum_multiple_same_bucket", false, []int64{10, 20, 30}, 60, "Max"},
		{"min_single", true, []int64{50}, 50, "Min"},
		{"min_multiple_same_bucket", true, []int64{100, 50, 200}, 50, "Min"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newRollingCounter(time.Second, 10, tt.isMin)
			now := time.Now()
			for _, v := range tt.adds {
				c.Add(now, v)
			}
			var got int64
			if tt.method == "Max" {
				got = c.Max(now)
			} else {
				got = c.Min(now)
			}
			if got != tt.want {
				t.Errorf("%s() = %d, want %d", tt.method, got, tt.want)
			}
		})
	}
}

func TestAdd_Multiple(t *testing.T) {
	tests := []struct {
		name   string
		isMin  bool
		adds   []int64
		want   int64
		method string
	}{
		{"sum_many_values", false, []int64{1, 2, 3, 4, 5}, 15, "Max"},
		{"min_many_values", true, []int64{5, 4, 3, 2, 1}, 1, "Min"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newRollingCounter(time.Second, 10, tt.isMin)
			now := time.Now()
			for _, v := range tt.adds {
				c.Add(now, v)
			}
			var got int64
			if tt.method == "Max" {
				got = c.Max(now)
			} else {
				got = c.Min(now)
			}
			if got != tt.want {
				t.Errorf("%s() = %d, want %d", tt.method, got, tt.want)
			}
		})
	}
}

func TestMax_SumMode(t *testing.T) {
	tests := []struct {
		name string
		adds []int64
		want int64
	}{
		{"single_value", []int64{10}, 10},
		{"multiple_values", []int64{10, 20, 5}, 35},
		{"zero_value", []int64{0}, 0},
		{"large_values", []int64{1000, 2000, 3000}, 6000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newRollingCounter(time.Second, 10, false)
			now := time.Now()
			for _, v := range tt.adds {
				c.Add(now, v)
			}
			if got := c.Max(now); got != tt.want {
				t.Errorf("Max() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMax_SumMode_EmptyReturnsZero(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"empty_counter_returns_zero"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newRollingCounter(time.Second, 10, false)
			if got := c.Max(time.Now()); got != 0 {
				t.Errorf("Max() = %d, want 0", got)
			}
		})
	}
}

func TestMin_MinMode(t *testing.T) {
	tests := []struct {
		name string
		adds []int64
		want int64
	}{
		{"single_value", []int64{100}, 100},
		{"multiple_values", []int64{100, 50, 200}, 50},
		{"same_values", []int64{10, 10, 10}, 10},
		{"large_range", []int64{1, 1000, 500}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newRollingCounter(time.Second, 10, true)
			now := time.Now()
			for _, v := range tt.adds {
				c.Add(now, v)
			}
			if got := c.Min(now); got != tt.want {
				t.Errorf("Min() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMin_MinMode_EmptyReturnsZero(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"empty_counter_returns_zero"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newRollingCounter(time.Second, 10, true)
			if got := c.Min(time.Now()); got != 0 {
				t.Errorf("Min() = %d, want 0", got)
			}
		})
	}
}

func TestMax_MinMode(t *testing.T) {
	tests := []struct {
		name string
		adds []int64
		want int64
	}{
		{"ignores_MaxInt64", []int64{100}, 100},
		{"multiple_values_in_same_bucket", []int64{50, 100, 75}, 50},
		{"empty_returns_zero", []int64{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newRollingCounter(time.Second, 10, true)
			now := time.Now()
			for _, v := range tt.adds {
				c.Add(now, v)
			}
			if got := c.Max(now); got != tt.want {
				t.Errorf("Max() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRotate(t *testing.T) {
	tests := []struct {
		name      string
		isMin     bool
		addNow    int64
		addLater  int64
		wantMax   int64
		wantMin   int64
	}{
		{"sum_expires_old", false, 100, 50, 50, 0},
		{"min_expires_old", true, 100, 50, 50, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newRollingCounter(time.Millisecond*100, 10, tt.isMin)
			now := time.Now()
			c.Add(now, tt.addNow)
			time.Sleep(time.Millisecond * 150)
			later := time.Now()
			c.Add(later, tt.addLater)
			if got := c.Max(later); got != tt.wantMax {
				t.Errorf("Max() = %d, want %d", got, tt.wantMax)
			}
		})
	}
}

func TestRotate_AllBuckets(t *testing.T) {
	tests := []struct {
		name   string
		isMin  bool
		addVal int64
		want   int64
	}{
		{"sum_all_cleared", false, 100, 0},
		{"min_all_cleared", true, 100, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newRollingCounter(time.Millisecond*50, 5, tt.isMin)
			now := time.Now()
			c.Add(now, tt.addVal)
			time.Sleep(time.Millisecond * 350)
			later := time.Now()
			if got := c.Max(later); got != tt.want {
				t.Errorf("Max() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRotate_MoreThanBuckets(t *testing.T) {
	tests := []struct {
		name   string
		isMin  bool
		addVal int64
		want   int64
	}{
		{"sum_long_delay", false, 100, 0},
		{"min_long_delay", true, 100, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newRollingCounter(time.Millisecond*50, 5, tt.isMin)
			now := time.Now()
			c.Add(now, tt.addVal)
			time.Sleep(time.Millisecond * 500)
			later := time.Now()
			if got := c.Max(later); got != tt.want {
				t.Errorf("Max() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRotate_WithinSameBucket(t *testing.T) {
	tests := []struct {
		name   string
		isMin  bool
		add1   int64
		add2   int64
		want   int64
		method string
	}{
		{"sum_not_rotated", false, 10, 20, 30, "Max"},
		{"min_not_rotated", true, 100, 50, 50, "Min"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newRollingCounter(time.Second, 10, tt.isMin)
			now := time.Now()
			c.Add(now, tt.add1)
			c.Add(now, tt.add2)
			var got int64
			if tt.method == "Max" {
				got = c.Max(now)
			} else {
				got = c.Min(now)
			}
			if got != tt.want {
				t.Errorf("%s() = %d, want %d", tt.method, got, tt.want)
			}
		})
	}
}

func TestRotate_PartialBuckets(t *testing.T) {
	tests := []struct {
		name   string
		isMin  bool
		add1   int64
		add2   int64
		want   int64
	}{
		{"sum_partial", false, 100, 50, 50},
		{"min_partial", true, 100, 50, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newRollingCounter(time.Millisecond*100, 10, tt.isMin)
			now := time.Now()
			c.Add(now, tt.add1)
			time.Sleep(time.Millisecond * 120)
			later := time.Now()
			c.Add(later, tt.add2)
			if got := c.Max(later); got != tt.want {
				t.Errorf("Max() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestConcurrent_Add(t *testing.T) {
	tests := []struct {
		name      string
		isMin     bool
		goroutine int
		addsEach  int
		wantAtLeast int64
	}{
		{"sum_100_goroutines", false, 100, 1, 1},
		{"min_100_goroutines", true, 100, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newRollingCounter(time.Second, 100, tt.isMin)
			var wg sync.WaitGroup
			for i := 0; i < tt.goroutine; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < tt.addsEach; j++ {
						c.Add(time.Now(), 1)
					}
				}()
			}
			wg.Wait()
			if got := c.Max(time.Now()); got < tt.wantAtLeast {
				t.Errorf("Max() = %d, want at least %d", got, tt.wantAtLeast)
			}
		})
	}
}

func TestConcurrent_AddAndRead(t *testing.T) {
	tests := []struct {
		name      string
		isMin     bool
		goroutine int
	}{
		{"sum_add_and_read", false, 50},
		{"min_add_and_read", true, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newRollingCounter(time.Second, 100, tt.isMin)
			var wg sync.WaitGroup
			for i := 0; i < tt.goroutine; i++ {
				wg.Add(2)
				go func() {
					defer wg.Done()
					c.Add(time.Now(), 1)
				}()
				go func() {
					defer wg.Done()
					_ = c.Max(time.Now())
				}()
			}
			wg.Wait()
		})
	}
}

func TestMax_SumMode_MultipleBuckets(t *testing.T) {
	tests := []struct {
		name   string
		adds   map[int]int64
		want   int64
	}{
		{"two_buckets", map[int]int64{0: 10, 1: 20}, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newRollingCounter(time.Second, 10, false)
			now := time.Now()
			for _, v := range tt.adds {
				c.Add(now, v)
			}
			if got := c.Max(now); got != tt.want {
				t.Errorf("Max() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMin_MinMode_MultipleBuckets(t *testing.T) {
	tests := []struct {
		name   string
		adds   []int64
		want   int64
	}{
		{"various_values", []int64{10, 5, 20, 3, 15}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newRollingCounter(time.Second, 10, true)
			now := time.Now()
			for _, v := range tt.adds {
				c.Add(now, v)
			}
			if got := c.Min(now); got != tt.want {
				t.Errorf("Min() = %d, want %d", got, tt.want)
			}
		})
	}
}
