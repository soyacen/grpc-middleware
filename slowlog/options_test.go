package slowlog

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultOptions(t *testing.T) {
	tests := []struct {
		name string
		want time.Duration
	}{
		{
			name: "default_slow_request_threshold_is_5s",
			want: 5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultOptions()
			assert.Equal(t, tt.want, got.SlowRequestThreshold)
		})
	}
}

func TestSlowRequestThreshold(t *testing.T) {
	tests := []struct {
		name      string
		threshold time.Duration
		want      time.Duration
	}{
		{"threshold_1s", time.Second, time.Second},
		{"threshold_10s", 10 * time.Second, 10 * time.Second},
		{"threshold_100ms", 100 * time.Millisecond, 100 * time.Millisecond},
		{"threshold_0", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()
			o.apply(SlowRequestThreshold(tt.threshold))
			assert.Equal(t, tt.want, o.SlowRequestThreshold)
		})
	}
}

func TestOptionsApply(t *testing.T) {
	tests := []struct {
		name string
		opts []Option
		want time.Duration
	}{
		{
			name: "single_option",
			opts: []Option{SlowRequestThreshold(time.Second)},
			want: time.Second,
		},
		{
			name: "multiple_options_last_wins",
			opts: []Option{
				SlowRequestThreshold(time.Second),
				SlowRequestThreshold(2 * time.Second),
			},
			want: 2 * time.Second,
		},
		{
			name: "no_options_uses_default",
			opts: []Option{},
			want: 5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()
			o.apply(tt.opts...)
			assert.Equal(t, tt.want, o.SlowRequestThreshold)
		})
	}
}
