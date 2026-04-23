package accesslog

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultOptions(t *testing.T) {
	tests := []struct {
		name     string
		wantLevel slog.Level
		wantSkip bool
	}{
		{
			name:      "default_options_values",
			wantLevel: slog.LevelInfo,
			wantSkip:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultOptions()
			assert.Equal(t, tt.wantLevel, got.level)
			assert.NotNil(t, got.skip)
			assert.False(t, got.skip("/test/method", nil))
		})
	}
}

func TestWithLevel(t *testing.T) {
	tests := []struct {
		name  string
		level slog.Level
		want  slog.Level
	}{
		{
			name:  "set_debug_level",
			level: slog.LevelDebug,
			want:  slog.LevelDebug,
		},
		{
			name:  "set_warn_level",
			level: slog.LevelWarn,
			want:  slog.LevelWarn,
		},
		{
			name:  "set_error_level",
			level: slog.LevelError,
			want:  slog.LevelError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()
			o.apply(WithLevel(tt.level))
			assert.Equal(t, tt.want, o.level)
		})
	}
}

func TestWithSkip(t *testing.T) {
	tests := []struct {
		name     string
		skip     func(string, error) bool
		method   string
		err      error
		wantSkip bool
	}{
		{
			name:     "skip_always_true",
			skip:     func(string, error) bool { return true },
			method:   "/test/method",
			err:      nil,
			wantSkip: true,
		},
		{
			name:     "skip_always_false",
			skip:     func(string, error) bool { return false },
			method:   "/test/method",
			err:      nil,
			wantSkip: false,
		},
		{
			name:     "skip_based_on_error",
			skip:     func(_ string, err error) bool { return err != nil },
			method:   "/test/method",
			err:      assert.AnError,
			wantSkip: true,
		},
		{
			name:     "skip_based_on_method",
			skip:     func(method string, _ error) bool { return method == "/skip/this" },
			method:   "/skip/this",
			err:      nil,
			wantSkip: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()
			o.apply(WithSkip(tt.skip))
			assert.Equal(t, tt.wantSkip, o.skip(tt.method, tt.err))
		})
	}
}

func TestOptionsApply(t *testing.T) {
	tests := []struct {
		name      string
		opts      []Option
		wantLevel slog.Level
		wantSkip  bool
	}{
		{
			name:      "single_option",
			opts:      []Option{WithLevel(slog.LevelDebug)},
			wantLevel: slog.LevelDebug,
			wantSkip:  false,
		},
		{
			name:      "multiple_options",
			opts:      []Option{WithLevel(slog.LevelDebug), WithSkip(func(string, error) bool { return true })},
			wantLevel: slog.LevelDebug,
			wantSkip:  true,
		},
		{
			name:      "no_options_uses_default",
			opts:      []Option{},
			wantLevel: slog.LevelInfo,
			wantSkip:  false,
		},
		{
			name:      "last_option_wins_for_level",
			opts:      []Option{WithLevel(slog.LevelDebug), WithLevel(slog.LevelError)},
			wantLevel: slog.LevelError,
			wantSkip:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()
			o.apply(tt.opts...)
			assert.Equal(t, tt.wantLevel, o.level)
			assert.Equal(t, tt.wantSkip, o.skip("/test/method", nil))
		})
	}
}
