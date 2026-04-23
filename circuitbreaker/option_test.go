package circuitbreaker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultOptions(t *testing.T) {
	tests := []struct {
		name       string
		wantK      float64
		wantWindow time.Duration
		wantBuckets int
	}{
		{
			name:       "default_k_is_2.0",
			wantK:      2.0,
			wantWindow: time.Second * 10,
			wantBuckets: 40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultOptions()
			assert.Equal(t, tt.wantK, got.K)
			assert.Equal(t, tt.wantWindow, got.Window)
			assert.Equal(t, tt.wantBuckets, got.Buckets)
		})
	}
}

func TestWithK(t *testing.T) {
	tests := []struct {
		name string
		k    float64
		want float64
	}{
		{"k_equals_1.0", 1.0, 1.0},
		{"k_equals_3.0", 3.0, 3.0},
		{"k_equals_5.5", 5.5, 5.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions().apply(WithK(tt.k))
			assert.Equal(t, tt.want, o.K)
		})
	}
}

func TestWithWindow(t *testing.T) {
	tests := []struct {
		name   string
		window time.Duration
		want   time.Duration
	}{
		{"window_5_seconds", time.Second * 5, time.Second * 5},
		{"window_30_seconds", time.Second * 30, time.Second * 30},
		{"window_1_minute", time.Minute, time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions().apply(WithWindow(tt.window))
			assert.Equal(t, tt.want, o.Window)
		})
	}
}

func TestWithBuckets(t *testing.T) {
	tests := []struct {
		name    string
		buckets int
		want    int
	}{
		{"buckets_10", 10, 10},
		{"buckets_20", 20, 20},
		{"buckets_100", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions().apply(WithBuckets(tt.buckets))
			assert.Equal(t, tt.want, o.Buckets)
		})
	}
}

func TestWithK_Zero(t *testing.T) {
	tests := []struct {
		name string
		k    float64
		want float64
	}{
		{"zero_k_set_by_apply", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions().apply(WithK(tt.k))
			assert.Equal(t, tt.want, o.K)
		})
	}
}

func TestWithK_Negative(t *testing.T) {
	tests := []struct {
		name string
		k    float64
		want float64
	}{
		{"negative_k_defaults_to_2.0", -1.0, 2.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &options{K: tt.k, Window: time.Second * 10, Buckets: 40}
			o.init()
			assert.Equal(t, tt.want, o.K)
		})
	}
}

func TestWithK_VeryLarge(t *testing.T) {
	tests := []struct {
		name string
		k    float64
		want float64
	}{
		{"very_large_k_accepted", 1e6, 1e6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions().apply(WithK(tt.k))
			assert.Equal(t, tt.want, o.K)
		})
	}
}

func TestApply_MultipleOptions(t *testing.T) {
	tests := []struct {
		name        string
		opts        []Option
		wantK       float64
		wantWindow  time.Duration
		wantBuckets int
	}{
		{
			name:        "apply_all_options",
			opts:        []Option{WithK(3.0), WithWindow(time.Second * 5), WithBuckets(20)},
			wantK:       3.0,
			wantWindow:  time.Second * 5,
			wantBuckets: 20,
		},
		{
			name:        "apply_in_reverse_order",
			opts:        []Option{WithBuckets(50), WithWindow(time.Minute), WithK(1.5)},
			wantK:       1.5,
			wantWindow:  time.Minute,
			wantBuckets: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions().apply(tt.opts...)
			assert.Equal(t, tt.wantK, o.K)
			assert.Equal(t, tt.wantWindow, o.Window)
			assert.Equal(t, tt.wantBuckets, o.Buckets)
		})
	}
}

func TestApply_Empty(t *testing.T) {
	tests := []struct {
		name        string
		opts        []Option
		wantK       float64
		wantWindow  time.Duration
		wantBuckets int
	}{
		{
			name:        "empty_options_keeps_defaults",
			opts:        []Option{},
			wantK:       2.0,
			wantWindow:  time.Second * 10,
			wantBuckets: 40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions().apply(tt.opts...)
			assert.Equal(t, tt.wantK, o.K)
			assert.Equal(t, tt.wantWindow, o.Window)
			assert.Equal(t, tt.wantBuckets, o.Buckets)
		})
	}
}

func TestInit_FixesInvalidK(t *testing.T) {
	tests := []struct {
		name string
		k    float64
		want float64
	}{
		{"negative_k", -5.0, 2.0},
		{"zero_k", 0.0, 2.0},
		{"positive_k_unchanged", 3.0, 3.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &options{K: tt.k, Window: time.Second * 10, Buckets: 40}
			o.init()
			assert.Equal(t, tt.want, o.K)
		})
	}
}

func TestInit_FixesInvalidWindow(t *testing.T) {
	tests := []struct {
		name   string
		window time.Duration
		want   time.Duration
	}{
		{"negative_window", -time.Second, time.Second * 10},
		{"zero_window", 0, time.Second * 10},
		{"positive_window_unchanged", time.Second * 5, time.Second * 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &options{K: 2.0, Window: tt.window, Buckets: 40}
			o.init()
			assert.Equal(t, tt.want, o.Window)
		})
	}
}

func TestInit_FixesInvalidBuckets(t *testing.T) {
	tests := []struct {
		name    string
		buckets int
		want    int
	}{
		{"negative_buckets", -10, 40},
		{"zero_buckets", 0, 40},
		{"positive_buckets_unchanged", 20, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &options{K: 2.0, Window: time.Second * 10, Buckets: tt.buckets}
			o.init()
			assert.Equal(t, tt.want, o.Buckets)
		})
	}
}

func TestNewCircuitBreaker(t *testing.T) {
	tests := []struct {
		name string
		opts []Option
		want bool
	}{
		{
			name: "new_with_defaults",
			opts: []Option{},
			want: true,
		},
		{
			name: "new_with_custom_options",
			opts: []Option{WithK(3.0), WithWindow(time.Second * 5), WithBuckets(20)},
			want: true,
		},
		{
			name: "new_with_invalid_options_gets_fixed",
			opts: []Option{WithK(-1.0), WithWindow(0), WithBuckets(0)},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions().apply(tt.opts...).init()
			cb := o.newCircuitBreaker()
			assert.NotNil(t, cb)
			assert.Implements(t, (*CircuitBreaker)(nil), cb)
		})
	}
}
