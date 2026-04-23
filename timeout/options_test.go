package timeout

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultOptions(t *testing.T) {
	tests := []struct {
		name     string
		expected time.Duration
	}{
		{"default_timeout_is_5_seconds", 5 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()
			assert.Equal(t, tt.expected, o.Timeout)
		})
	}
}

func TestTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		want    time.Duration
	}{
		{"timeout_1_second", time.Second, time.Second},
		{"timeout_3_seconds", 3 * time.Second, 3 * time.Second},
		{"timeout_10_seconds", 10 * time.Second, 10 * time.Second},
		{"timeout_100_milliseconds", 100 * time.Millisecond, 100 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions().apply(Timeout(tt.timeout))
			assert.Equal(t, tt.want, o.Timeout)
		})
	}
}

func TestTimeout_Zero(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		want    time.Duration
	}{
		{"zero_timeout", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions().apply(Timeout(tt.timeout))
			assert.Equal(t, tt.want, o.Timeout)
		})
	}
}

func TestTimeout_Negative(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		want    time.Duration
	}{
		{"negative_timeout", -time.Second, -time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions().apply(Timeout(tt.timeout))
			assert.Equal(t, tt.want, o.Timeout)
		})
	}
}

func TestApply_MultipleOptions(t *testing.T) {
	tests := []struct {
		name     string
		opts     []Option
		expected time.Duration
	}{
		{
			name:     "apply_single_option",
			opts:     []Option{Timeout(3 * time.Second)},
			expected: 3 * time.Second,
		},
		{
			name:     "apply_multiple_options_overwrite",
			opts:     []Option{Timeout(time.Second), Timeout(10 * time.Second)},
			expected: 10 * time.Second,
		},
		{
			name:     "apply_in_reverse_order",
			opts:     []Option{Timeout(10 * time.Second), Timeout(time.Second)},
			expected: time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions().apply(tt.opts...)
			assert.Equal(t, tt.expected, o.Timeout)
		})
	}
}

func TestApply_Empty(t *testing.T) {
	tests := []struct {
		name     string
		opts     []Option
		expected time.Duration
	}{
		{
			name:     "empty_options_keeps_defaults",
			opts:     []Option{},
			expected: 5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions().apply(tt.opts...)
			assert.Equal(t, tt.expected, o.Timeout)
		})
	}
}
