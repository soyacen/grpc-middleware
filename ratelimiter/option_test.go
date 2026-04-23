package ratelimiter

import (
	"context"
	"testing"
	"time"
)

func TestDefaultOptions(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		expected interface{}
	}{
		{"Window_default", "Window", time.Second * 10},
		{"Buckets_default", "Buckets", 100},
		{"CPUThreshold_default", "CPUThreshold", 0.8},
		{"CPUInterval_default", "CPUInterval", time.Millisecond * 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()
			switch tt.field {
			case "Window":
				if o.Window != tt.expected.(time.Duration) {
					t.Errorf("Window = %v, want %v", o.Window, tt.expected)
				}
			case "Buckets":
				if o.Buckets != tt.expected.(int) {
					t.Errorf("Buckets = %v, want %v", o.Buckets, tt.expected)
				}
			case "CPUThreshold":
				if o.CPUThreshold != tt.expected.(float64) {
					t.Errorf("CPUThreshold = %v, want %v", o.CPUThreshold, tt.expected)
				}
			case "CPUInterval":
				if o.CPUInterval != tt.expected.(time.Duration) {
					t.Errorf("CPUInterval = %v, want %v", o.CPUInterval, tt.expected)
				}
			}
		})
	}
}

func TestWithWindow(t *testing.T) {
	tests := []struct {
		name   string
		window time.Duration
		want   time.Duration
	}{
		{"5_seconds", 5 * time.Second, 5 * time.Second},
		{"1_minute", time.Minute, time.Minute},
		{"100_milliseconds", 100 * time.Millisecond, 100 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &options{}
			WithWindow(tt.window)(o)
			if o.Window != tt.want {
				t.Errorf("Window = %v, want %v", o.Window, tt.want)
			}
		})
	}
}

func TestWithBuckets(t *testing.T) {
	tests := []struct {
		name    string
		buckets int
		want    int
	}{
		{"10_buckets", 10, 10},
		{"50_buckets", 50, 50},
		{"200_buckets", 200, 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &options{}
			WithBuckets(tt.buckets)(o)
			if o.Buckets != tt.want {
				t.Errorf("Buckets = %v, want %v", o.Buckets, tt.want)
			}
		})
	}
}

func TestWithCPUThreshold(t *testing.T) {
	tests := []struct {
		name     string
		threshold float64
		want     float64
	}{
		{"threshold_0.5", 0.5, 0.5},
		{"threshold_0.9", 0.9, 0.9},
		{"threshold_1.0", 1.0, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &options{}
			WithCPUThreshold(tt.threshold)(o)
			if o.CPUThreshold != tt.want {
				t.Errorf("CPUThreshold = %v, want %v", o.CPUThreshold, tt.want)
			}
		})
	}
}

func TestWithCPUThreshold_Zero(t *testing.T) {
	tests := []struct {
		name     string
		threshold float64
		want     float64
	}{
		{"zero_threshold", 0, 0},
		{"negative_threshold", -0.1, -0.1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &options{}
			WithCPUThreshold(tt.threshold)(o)
			if o.CPUThreshold != tt.want {
				t.Errorf("CPUThreshold = %v, want %v", o.CPUThreshold, tt.want)
			}
		})
	}
}

func TestWithCPUThreshold_AboveOne(t *testing.T) {
	tests := []struct {
		name     string
		threshold float64
		want     float64
	}{
		{"threshold_1.5", 1.5, 1.5},
		{"threshold_2.0", 2.0, 2.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &options{}
			WithCPUThreshold(tt.threshold)(o)
			if o.CPUThreshold != tt.want {
				t.Errorf("CPUThreshold = %v, want %v", o.CPUThreshold, tt.want)
			}
		})
	}
}

func TestWithCPU(t *testing.T) {
	tests := []struct {
		name string
		cpu  func() float64
		want float64
	}{
		{"cpu_returns_0.5", func() float64 { return 0.5 }, 0.5},
		{"cpu_returns_0.9", func() float64 { return 0.9 }, 0.9},
		{"cpu_returns_0.0", func() float64 { return 0.0 }, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &options{}
			WithCPU(tt.cpu)(o)
			if o.CPU == nil {
				t.Fatal("CPU function should not be nil")
			}
			if got := o.CPU(); got != tt.want {
				t.Errorf("CPU() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithCPU_Nil(t *testing.T) {
	tests := []struct {
		name string
		cpu  func() float64
	}{
		{"nil_cpu", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &options{}
			WithCPU(tt.cpu)(o)
			if o.CPU != nil {
				t.Error("CPU should be nil")
			}
		})
	}
}

func TestWithCPU_Injection(t *testing.T) {
	tests := []struct {
		name     string
		cpuFunc  func() float64
		expected float64
	}{
		{"injected_cpu_func", func() float64 { return 0.75 }, 0.75},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()
			WithCPU(tt.cpuFunc)(o)
			if o.CPU() != tt.expected {
				t.Errorf("CPU() = %v, want %v", o.CPU(), tt.expected)
			}
		})
	}
}

func TestWithCPUInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval time.Duration
		want     time.Duration
	}{
		{"250_milliseconds", 250 * time.Millisecond, 250 * time.Millisecond},
		{"1_second", time.Second, time.Second},
		{"2_seconds", 2 * time.Second, 2 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &options{}
			WithCPUInterval(tt.interval)(o)
			if o.CPUInterval != tt.want {
				t.Errorf("CPUInterval = %v, want %v", o.CPUInterval, tt.want)
			}
		})
	}
}

func TestWithSkip(t *testing.T) {
	tests := []struct {
		name     string
		skip     func(ctx context.Context, fullMethod string) bool
		expected bool
	}{
		{"skip_true", func(ctx context.Context, fullMethod string) bool { return true }, true},
		{"skip_false", func(ctx context.Context, fullMethod string) bool { return false }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &options{}
			WithSkip(tt.skip)(o)
			if o.Skip == nil {
				t.Fatal("Skip function should not be nil")
			}
			if got := o.Skip(context.Background(), "/test.Service/Method"); got != tt.expected {
				t.Errorf("Skip() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestWithSkip_True(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"skip_returns_true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &options{}
			WithSkip(func(ctx context.Context, fullMethod string) bool { return true })(o)
			if o.Skip == nil || !o.Skip(context.Background(), "/test.Service/Method") {
				t.Error("Skip should return true")
			}
		})
	}
}

func TestWithSkip_False(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"skip_returns_false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &options{}
			WithSkip(func(ctx context.Context, fullMethod string) bool { return false })(o)
			if o.Skip == nil || o.Skip(context.Background(), "/test.Service/Method") {
				t.Error("Skip should return false")
			}
		})
	}
}

func TestWithSkip_Nil(t *testing.T) {
	tests := []struct {
		name string
		skip func(ctx context.Context, fullMethod string) bool
	}{
		{"nil_skip", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &options{}
			WithSkip(tt.skip)(o)
			if o.Skip != nil {
				t.Error("Skip should be nil")
			}
		})
	}
}

func TestInit_FixesInvalidValues(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*options)
		check    func(*options) bool
		expected interface{}
	}{
		{
			name: "fix_zero_Window",
			setup: func(o *options) { o.Window = 0 },
			check: func(o *options) bool { return o.Window == time.Second*10 },
		},
		{
			name: "fix_negative_Window",
			setup: func(o *options) { o.Window = -1 * time.Second },
			check: func(o *options) bool { return o.Window == time.Second*10 },
		},
		{
			name: "fix_zero_Buckets",
			setup: func(o *options) { o.Buckets = 0 },
			check: func(o *options) bool { return o.Buckets == 100 },
		},
		{
			name: "fix_negative_Buckets",
			setup: func(o *options) { o.Buckets = -5 },
			check: func(o *options) bool { return o.Buckets == 100 },
		},
		{
			name: "fix_zero_CPUThreshold",
			setup: func(o *options) { o.CPUThreshold = 0 },
			check: func(o *options) bool { return o.CPUThreshold == 0.8 },
		},
		{
			name: "fix_negative_CPUThreshold",
			setup: func(o *options) { o.CPUThreshold = -0.5 },
			check: func(o *options) bool { return o.CPUThreshold == 0.8 },
		},
		{
			name: "fix_zero_CPUInterval",
			setup: func(o *options) { o.CPUInterval = 0 },
			check: func(o *options) bool { return o.CPUInterval == time.Millisecond*500 },
		},
		{
			name: "fix_nil_CPU",
			setup: func(o *options) { o.CPU = nil },
			check: func(o *options) bool { return o.CPU != nil },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &options{}
			tt.setup(o)
			o.init()
			if !tt.check(o) {
				t.Error("init did not fix invalid value correctly")
			}
		})
	}
}

func TestApply(t *testing.T) {
	tests := []struct {
		name     string
		opts     []Option
		check    func(*options) bool
	}{
		{
			name: "apply_single_option",
			opts: []Option{WithWindow(5 * time.Second)},
			check: func(o *options) bool { return o.Window == 5*time.Second },
		},
		{
			name: "apply_multiple_options",
			opts: []Option{
				WithWindow(5 * time.Second),
				WithBuckets(50),
				WithCPUThreshold(0.9),
			},
			check: func(o *options) bool {
				return o.Window == 5*time.Second && o.Buckets == 50 && o.CPUThreshold == 0.9
			},
		},
		{
			name: "apply_no_options",
			opts: []Option{},
			check: func(o *options) bool { return o.Window == time.Second*10 },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()
			o.apply(tt.opts...)
			if !tt.check(o) {
				t.Error("apply did not apply options correctly")
			}
		})
	}
}
