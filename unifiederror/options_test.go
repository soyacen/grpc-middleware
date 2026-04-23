package unifiederror

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestDefaultOptions(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"default_options_created"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()
			assert.NotNil(t, o)
			assert.NotNil(t, o.errorFunc)
		})
	}
}

func TestDefaultOptions_ErrorFunc(t *testing.T) {
	tests := []struct {
		name        string
		inputErr    error
		wantNil     bool
		wantCode    codes.Code
		wantMessage string
	}{
		{
			name:     "nil_error_returns_nil",
			inputErr: nil,
			wantNil:  true,
		},
		{
			name:        "deadline_exceeded",
			inputErr:    errors.New("context deadline exceeded"),
			wantCode:    codes.Unknown,
			wantMessage: "context deadline exceeded",
		},
		{
			name:        "plain_error",
			inputErr:    errors.New("something went wrong"),
			wantCode:    codes.Unknown,
			wantMessage: "something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()
			result := o.errorFunc(tt.inputErr)

			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.wantCode, result.Code())
				assert.Equal(t, tt.wantMessage, result.Message())
			}
		})
	}
}

func TestErrorFunc(t *testing.T) {
	tests := []struct {
		name         string
		customFunc   func(err error) *status.Status
		inputErr     error
		wantNil      bool
		wantCode     codes.Code
		wantMessage  string
	}{
		{
			name: "custom_func_returns_nil",
			customFunc: func(err error) *status.Status {
				return nil
			},
			inputErr: nil,
			wantNil:  true,
		},
		{
			name: "custom_func_converts_to_internal",
			customFunc: func(err error) *status.Status {
				if err == nil {
					return nil
				}
				return status.New(codes.Internal, "internal: "+err.Error())
			},
			inputErr:    errors.New("failure"),
			wantCode:    codes.Internal,
			wantMessage: "internal: failure",
		},
		{
			name: "custom_func_converts_all_to_ok",
			customFunc: func(err error) *status.Status {
				return status.New(codes.OK, "always ok")
			},
			inputErr:    errors.New("any error"),
			wantCode:    codes.OK,
			wantMessage: "always ok",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()
			opt := ErrorFunc(tt.customFunc)
			opt(o)

			result := o.errorFunc(tt.inputErr)

			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.wantCode, result.Code())
				assert.Equal(t, tt.wantMessage, result.Message())
			}
		})
	}
}

func TestApply(t *testing.T) {
	tests := []struct {
		name       string
		opts       []Option
		wantCode   codes.Code
		wantMsg    string
	}{
		{
			name:    "empty_options",
			opts:    []Option{},
			wantCode: codes.Unknown,
			wantMsg:  "test",
		},
		{
			name: "single_option",
			opts: []Option{
				ErrorFunc(func(err error) *status.Status {
					return status.New(codes.NotFound, "not found")
				}),
			},
			wantCode: codes.NotFound,
			wantMsg:  "not found",
		},
		{
			name: "multiple_options_last_wins",
			opts: []Option{
				ErrorFunc(func(err error) *status.Status {
					return status.New(codes.Internal, "first")
				}),
				ErrorFunc(func(err error) *status.Status {
					return status.New(codes.Unavailable, "second")
				}),
			},
			wantCode: codes.Unavailable,
			wantMsg:  "second",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()
			o.apply(tt.opts...)

			result := o.errorFunc(errors.New("test"))

			assert.NotNil(t, result)
			assert.Equal(t, tt.wantCode, result.Code())
			assert.Equal(t, tt.wantMsg, result.Message())
		})
	}
}
