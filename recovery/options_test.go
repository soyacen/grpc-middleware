package recovery

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultOptions(t *testing.T) {
	o := defaultOptions()
	assert.NotNil(t, o)
	assert.NotNil(t, o.handler)
}

func TestRecoveryHandler(t *testing.T) {
	customCalled := false
	customHandler := func(ctx context.Context, method string, p any) error {
		customCalled = true
		return assert.AnError
	}

	o := defaultOptions().apply(RecoveryHandler(customHandler))
	assert.NotNil(t, o.handler)

	err := o.handler(context.Background(), "/test.Service/Method", "panic")
	assert.Equal(t, assert.AnError, err)
	assert.True(t, customCalled)
}

func TestApply_MultipleOptions(t *testing.T) {
	handler1Called := false
	handler2Called := false

	handler1 := func(ctx context.Context, method string, p any) error {
		handler1Called = true
		return nil
	}
	handler2 := func(ctx context.Context, method string, p any) error {
		handler2Called = true
		return nil
	}

	o := defaultOptions().apply(RecoveryHandler(handler1), RecoveryHandler(handler2))
	o.handler(context.Background(), "/test.Service/Method", "panic")
	assert.False(t, handler1Called)
	assert.True(t, handler2Called)
}

func TestApply_Empty(t *testing.T) {
	o := defaultOptions().apply()
	assert.NotNil(t, o.handler)

	err := o.handler(context.Background(), "/test.Service/Method", "panic")
	assert.Error(t, err)

	var panicErr *PanicError
	assert.ErrorAs(t, err, &panicErr)
	assert.Equal(t, "/test.Service/Method", panicErr.Method)
	assert.Equal(t, "panic", panicErr.Panic)
	assert.NotNil(t, panicErr.Stack)
}

func TestDefaultHandler(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		panicVal any
	}{
		{"string_panic", "/test.Service/Method", "panic message"},
		{"error_panic", "/test.Service/Method2", errors.New("panic error")},
		{"int_panic", "/test.Service/Method3", 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := defaultHandler(context.Background(), tt.method, tt.panicVal)
			assert.Error(t, err)

			var panicErr *PanicError
			assert.ErrorAs(t, err, &panicErr)
			assert.Equal(t, tt.method, panicErr.Method)
			assert.Equal(t, tt.panicVal, panicErr.Panic)
			assert.NotNil(t, panicErr.Stack)
			assert.NotEmpty(t, panicErr.Stack)
		})
	}
}
