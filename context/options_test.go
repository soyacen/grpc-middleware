package context

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultOptions(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"default_options"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()
			assert.NotNil(t, o)
			assert.NotNil(t, o.contextFunc)

			ctx := context.WithValue(context.Background(), "key", "value")
			result := o.contextFunc(ctx)
			assert.Equal(t, ctx, result)
			assert.Equal(t, "value", result.Value("key"))
		})
	}
}

func TestWithContextFunc(t *testing.T) {
	tests := []struct {
		name        string
		contextFunc ContextFunc
		wantKey     string
		wantValue   string
	}{
		{
			name: "add_value_to_context",
			contextFunc: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, "test", "value")
			},
			wantKey:   "test",
			wantValue: "value",
		},
		{
			name: "override_existing_value",
			contextFunc: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, "existing", "overridden")
			},
			wantKey:   "existing",
			wantValue: "overridden",
		},
		{
			name: "return_same_context",
			contextFunc: func(ctx context.Context) context.Context {
				return ctx
			},
			wantKey:   "key",
			wantValue: "original",
		},
		{
			name: "create_cancel_context",
			contextFunc: func(ctx context.Context) context.Context {
				newCtx, cancel := context.WithCancel(ctx)
				_ = cancel
				return newCtx
			},
			wantKey:   "key",
			wantValue: "original",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()
			WithContextFunc(tt.contextFunc)(o)
			assert.NotNil(t, o.contextFunc)

			ctx := context.WithValue(context.Background(), "key", "original")
			if tt.name == "override_existing_value" {
				ctx = context.WithValue(ctx, "existing", "original")
			}

			result := o.contextFunc(ctx)
			assert.Equal(t, tt.wantValue, result.Value(tt.wantKey))
		})
	}
}

func TestWithContextFunc_Nil(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"nil_context_func"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()
			WithContextFunc(nil)(o)
			assert.Nil(t, o.contextFunc)
		})
	}
}

func TestApply(t *testing.T) {
	tests := []struct {
		name        string
		opts        []Option
		wantKey     string
		wantValue   string
	}{
		{
			name:        "single_option",
			opts:        []Option{WithContextFunc(func(ctx context.Context) context.Context {
				return context.WithValue(ctx, "single", "1")
			})},
			wantKey:     "single",
			wantValue:   "1",
		},
		{
			name:        "multiple_options",
			opts:        []Option{
				WithContextFunc(func(ctx context.Context) context.Context {
					return context.WithValue(ctx, "first", "1")
				}),
				WithContextFunc(func(ctx context.Context) context.Context {
					return context.WithValue(ctx, "second", "2")
				}),
			},
			wantKey:     "second",
			wantValue:   "2",
		},
		{
			name:        "empty_options",
			opts:        []Option{},
			wantKey:     "key",
			wantValue:   "original",
		},

	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()
			o.apply(tt.opts...)
			assert.NotNil(t, o.contextFunc)

			ctx := context.WithValue(context.Background(), "key", "original")
			result := o.contextFunc(ctx)
			assert.Equal(t, tt.wantValue, result.Value(tt.wantKey))
		})
	}
}

func TestApply_Override(t *testing.T) {
	tests := []struct {
		name      string
		first     ContextFunc
		second    ContextFunc
		wantKey   string
		wantValue string
	}{
		{
			name: "second_overrides_first",
			first: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, "key", "first")
			},
			second: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, "key", "second")
			},
			wantKey:   "key",
			wantValue: "second",
		},
		{
			name: "second_different_key",
			first: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, "first", "1")
			},
			second: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, "second", "2")
			},
			wantKey:   "second",
			wantValue: "2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := defaultOptions()
			o.apply(
				WithContextFunc(tt.first),
				WithContextFunc(tt.second),
			)

			ctx := context.Background()
			result := o.contextFunc(ctx)
			assert.Equal(t, tt.wantValue, result.Value(tt.wantKey))
		})
	}
}

func TestOptionsStruct(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"options_struct_creation"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &options{
				contextFunc: func(ctx context.Context) context.Context {
					return context.WithValue(ctx, "custom", "value")
				},
			}

			ctx := context.Background()
			result := o.contextFunc(ctx)
			assert.Equal(t, "value", result.Value("custom"))
		})
	}
}
