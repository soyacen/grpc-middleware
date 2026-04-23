package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/soyacen/gox/backoff"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestDefaultOptions(t *testing.T) {
	tests := []struct {
		name string
		want *options
	}{
		{
			name: "default_values",
			want: &options{
				MaxRetries:     3,
				BackoffFunc:    backoff.Exponential2(100 * time.Millisecond),
				RetryableFunc:  defaultRetryableFunc,
				PerCallTimeout: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultOptions()
			assert.Equal(t, tt.want.MaxRetries, got.MaxRetries)
			assert.NotNil(t, got.BackoffFunc)
			assert.NotNil(t, got.RetryableFunc)
			assert.Equal(t, tt.want.PerCallTimeout, got.PerCallTimeout)
		})
	}
}

func TestMaxRetries(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want int
	}{
		{"positive_retries", 5, 5},
		{"zero_retries", 0, 0},
		{"negative_retries_ignored", -1, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions().apply(MaxRetries(tt.n))
			assert.Equal(t, tt.want, o.MaxRetries)
		})
	}
}

func TestBackoffFunc(t *testing.T) {
	customBackoff := func(ctx context.Context, attempt uint) time.Duration {
		return time.Duration(attempt) * time.Second
	}

	tests := []struct {
		name string
		fn   backoff.Func
		want time.Duration
	}{
		{"custom_func", customBackoff, time.Second},
		{"nil_func_ignored", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions().apply(BackoffFunc(tt.fn))
			assert.NotNil(t, o.BackoffFunc)
			if tt.fn != nil {
				got := o.BackoffFunc(context.Background(), 1)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestWithRetryableFunc(t *testing.T) {
	customRetryable := func(err error) bool {
		return err != nil && err.Error() == "retry me"
	}

	tests := []struct {
		name     string
		fn       func(error) bool
		wantTrue bool
	}{
		{"custom_func", customRetryable, true},
		{"nil_func_ignored", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions().apply(WithRetryableFunc(tt.fn))
			assert.NotNil(t, o.RetryableFunc)
			if tt.fn != nil {
				err := errors.New("retry me")
				got := o.RetryableFunc(err)
				assert.Equal(t, tt.wantTrue, got)
			}
		})
	}
}

func TestPerCallTimeout(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want time.Duration
	}{
		{"100_milliseconds", 100 * time.Millisecond, 100 * time.Millisecond},
		{"1_second", time.Second, time.Second},
		{"5_seconds", 5 * time.Second, 5 * time.Second},
		{"zero_duration", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions().apply(PerCallTimeout(tt.d))
			assert.Equal(t, tt.want, o.PerCallTimeout)
		})
	}
}

func TestApply_MultipleOptions(t *testing.T) {
	customBackoff := func(ctx context.Context, attempt uint) time.Duration {
		return time.Second
	}
	customRetryable := func(err error) bool {
		return false
	}

	tests := []struct {
		name           string
		opts           []Option
		wantMaxRetries int
		wantTimeout    time.Duration
	}{
		{
			name:           "apply_all_options",
			opts:           []Option{MaxRetries(5), BackoffFunc(customBackoff), WithRetryableFunc(customRetryable), PerCallTimeout(time.Second)},
			wantMaxRetries: 5,
			wantTimeout:    time.Second,
		},
		{
			name:           "apply_in_reverse_order",
			opts:           []Option{PerCallTimeout(2 * time.Second), WithRetryableFunc(customRetryable), BackoffFunc(customBackoff), MaxRetries(10)},
			wantMaxRetries: 10,
			wantTimeout:    2 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions().apply(tt.opts...)
			assert.Equal(t, tt.wantMaxRetries, o.MaxRetries)
			assert.Equal(t, tt.wantTimeout, o.PerCallTimeout)
		})
	}
}

func TestApply_Empty(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"empty_options_keeps_defaults"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions().apply()
			assert.Equal(t, 3, o.MaxRetries)
			assert.NotNil(t, o.BackoffFunc)
			assert.NotNil(t, o.RetryableFunc)
			assert.Equal(t, time.Duration(0), o.PerCallTimeout)
		})
	}
}

func TestDefaultRetryableFunc(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil_error", nil, false},
		{"non_status_error", errTest, false},
		{"codes_OK", status.Error(codes.OK, "ok"), false},
		{"codes_Cancelled", status.Error(codes.Canceled, "cancelled"), false},
		{"codes_Unknown", status.Error(codes.Unknown, "unknown"), false},
		{"codes_InvalidArgument", status.Error(codes.InvalidArgument, "invalid argument"), false},
		{"codes_DeadlineExceeded", status.Error(codes.DeadlineExceeded, "deadline exceeded"), true},
		{"codes_NotFound", status.Error(codes.NotFound, "not found"), false},
		{"codes_AlreadyExists", status.Error(codes.AlreadyExists, "already exists"), false},
		{"codes_PermissionDenied", status.Error(codes.PermissionDenied, "permission denied"), false},
		{"codes_ResourceExhausted", status.Error(codes.ResourceExhausted, "resource exhausted"), true},
		{"codes_FailedPrecondition", status.Error(codes.FailedPrecondition, "failed precondition"), false},
		{"codes_Aborted", status.Error(codes.Aborted, "aborted"), true},
		{"codes_OutOfRange", status.Error(codes.OutOfRange, "out of range"), false},
		{"codes_Unimplemented", status.Error(codes.Unimplemented, "unimplemented"), false},
		{"codes_Internal", status.Error(codes.Internal, "internal"), false},
		{"codes_Unavailable", status.Error(codes.Unavailable, "unavailable"), true},
		{"codes_DataLoss", status.Error(codes.DataLoss, "data loss"), false},
		{"codes_Unauthenticated", status.Error(codes.Unauthenticated, "unauthenticated"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultRetryableFunc(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

var errTest = status.Error(codes.Unknown, "test error")
