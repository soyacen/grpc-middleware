package context

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type mockServerStream struct {
	ctx context.Context
}

func (m *mockServerStream) SetHeader(md metadata.MD) error  { return nil }
func (m *mockServerStream) SendHeader(md metadata.MD) error { return nil }
func (m *mockServerStream) SetTrailer(md metadata.MD)       {}
func (m *mockServerStream) Context() context.Context        { return m.ctx }
func (m *mockServerStream) SendMsg(v interface{}) error     { return nil }
func (m *mockServerStream) RecvMsg(v interface{}) error     { return nil }

type mockClientStream struct {
	grpc.ClientStream
}

func TestUnaryServerInterceptor(t *testing.T) {
	tests := []struct {
		name        string
		contextFunc ContextFunc
		wantKey     string
		wantValue   string
		handlerErr  error
		wantErr     error
	}{
		{
			name: "default_context_func_no_change",
			contextFunc: func(ctx context.Context) context.Context {
				return ctx
			},
			wantKey:   "key",
			wantValue: "original",
			handlerErr: nil,
			wantErr:    nil,
		},
		{
			name: "custom_context_func_adds_value",
			contextFunc: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, "injected", "value")
			},
			wantKey:   "injected",
			wantValue: "value",
			handlerErr: nil,
			wantErr:    nil,
		},
		{
			name: "handler_returns_error",
			contextFunc: func(ctx context.Context) context.Context {
				return ctx
			},
			wantKey:   "key",
			wantValue: "original",
			handlerErr: errors.New("handler error"),
			wantErr:    errors.New("handler error"),
		},
		{
			name: "context_func_overrides_existing_value",
			contextFunc: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, "key", "overridden")
			},
			wantKey:   "key",
			wantValue: "overridden",
			handlerErr: nil,
			wantErr:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := UnaryServerInterceptor(WithContextFunc(tt.contextFunc))
			ctx := context.WithValue(context.Background(), "key", "original")
			var receivedCtx context.Context

			handler := func(c context.Context, req interface{}) (interface{}, error) {
				receivedCtx = c
				return "response", tt.handlerErr
			}

			resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "response", resp)
			}
			assert.Equal(t, tt.wantValue, receivedCtx.Value(tt.wantKey))
		})
	}
}

func TestUnaryServerInterceptor_ContextPropagation(t *testing.T) {
	interceptor := UnaryServerInterceptor(WithContextFunc(func(ctx context.Context) context.Context {
		return context.WithValue(ctx, "propagated", "yes")
	}))

	ctx := context.WithValue(context.Background(), "original", "value")
	var receivedCtx context.Context

	handler := func(c context.Context, req interface{}) (interface{}, error) {
		receivedCtx = c
		return nil, nil
	}

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)

	assert.NoError(t, err)
	assert.Equal(t, "value", receivedCtx.Value("original"))
	assert.Equal(t, "yes", receivedCtx.Value("propagated"))
}

func TestStreamServerInterceptor(t *testing.T) {
	tests := []struct {
		name        string
		contextFunc ContextFunc
		wantKey     string
		wantValue   string
		handlerErr  error
		wantErr     error
	}{
		{
			name: "default_context_func_no_change",
			contextFunc: func(ctx context.Context) context.Context {
				return ctx
			},
			wantKey:   "key",
			wantValue: "original",
			handlerErr: nil,
			wantErr:    nil,
		},
		{
			name: "custom_context_func_adds_value",
			contextFunc: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, "injected", "value")
			},
			wantKey:   "injected",
			wantValue: "value",
			handlerErr: nil,
			wantErr:    nil,
		},
		{
			name: "handler_returns_error",
			contextFunc: func(ctx context.Context) context.Context {
				return ctx
			},
			wantKey:   "key",
			wantValue: "original",
			handlerErr: errors.New("stream error"),
			wantErr:    errors.New("stream error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := StreamServerInterceptor(WithContextFunc(tt.contextFunc))
			ctx := context.WithValue(context.Background(), "key", "original")
			stream := &mockServerStream{ctx: ctx}
			var receivedCtx context.Context

			handler := func(srv interface{}, s grpc.ServerStream) error {
				receivedCtx = s.Context()
				return tt.handlerErr
			}

			err := interceptor(nil, stream, &grpc.StreamServerInfo{}, handler)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.NotNil(t, receivedCtx)
		})
	}
}

func TestStreamServerInterceptor_ContextModified(t *testing.T) {
	interceptor := StreamServerInterceptor(WithContextFunc(func(ctx context.Context) context.Context {
		return context.WithValue(ctx, "modified", "yes")
	}))

	ctx := context.WithValue(context.Background(), "original", "value")
	stream := &mockServerStream{ctx: ctx}

	var receivedCtx context.Context
	handler := func(srv interface{}, s grpc.ServerStream) error {
		receivedCtx = s.Context()
		return nil
	}

	err := interceptor(nil, stream, &grpc.StreamServerInfo{}, handler)

	assert.NoError(t, err)
	assert.Equal(t, "value", receivedCtx.Value("original"))
}

func TestUnaryClientInterceptor(t *testing.T) {
	tests := []struct {
		name        string
		contextFunc ContextFunc
		wantKey     string
		wantValue   string
		invokerErr  error
		wantErr     error
	}{
		{
			name: "default_context_func_no_change",
			contextFunc: func(ctx context.Context) context.Context {
				return ctx
			},
			wantKey:    "key",
			wantValue:  "original",
			invokerErr: nil,
			wantErr:    nil,
		},
		{
			name: "custom_context_func_adds_value",
			contextFunc: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, "injected", "value")
			},
			wantKey:    "injected",
			wantValue:  "value",
			invokerErr: nil,
			wantErr:    nil,
		},
		{
			name: "invoker_returns_error",
			contextFunc: func(ctx context.Context) context.Context {
				return ctx
			},
			wantKey:    "key",
			wantValue:  "original",
			invokerErr: errors.New("invoker error"),
			wantErr:    errors.New("invoker error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := UnaryClientInterceptor(WithContextFunc(tt.contextFunc))
			ctx := context.WithValue(context.Background(), "key", "original")
			var receivedCtx context.Context

			invoker := func(c context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				receivedCtx = c
				return tt.invokerErr
			}

			err := interceptor(ctx, "/test/method", nil, nil, nil, invoker)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantValue, receivedCtx.Value(tt.wantKey))
		})
	}
}

func TestUnaryClientInterceptor_ContextPropagation(t *testing.T) {
	interceptor := UnaryClientInterceptor(WithContextFunc(func(ctx context.Context) context.Context {
		return context.WithValue(ctx, "propagated", "yes")
	}))

	ctx := context.WithValue(context.Background(), "original", "value")
	var receivedCtx context.Context

	invoker := func(c context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		receivedCtx = c
		return nil
	}

	err := interceptor(ctx, "/test/method", nil, nil, nil, invoker)

	assert.NoError(t, err)
	assert.Equal(t, "value", receivedCtx.Value("original"))
	assert.Equal(t, "yes", receivedCtx.Value("propagated"))
}

func TestStreamClientInterceptor(t *testing.T) {
	tests := []struct {
		name         string
		contextFunc  ContextFunc
		wantKey      string
		wantValue    string
		streamerErr  error
		wantErr      error
	}{
		{
			name: "default_context_func_no_change",
			contextFunc: func(ctx context.Context) context.Context {
				return ctx
			},
			wantKey:     "key",
			wantValue:   "original",
			streamerErr: nil,
			wantErr:     nil,
		},
		{
			name: "custom_context_func_adds_value",
			contextFunc: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, "injected", "value")
			},
			wantKey:     "injected",
			wantValue:   "value",
			streamerErr: nil,
			wantErr:     nil,
		},
		{
			name: "streamer_returns_error",
			contextFunc: func(ctx context.Context) context.Context {
				return ctx
			},
			wantKey:     "key",
			wantValue:   "original",
			streamerErr: errors.New("streamer error"),
			wantErr:     errors.New("streamer error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := StreamClientInterceptor(WithContextFunc(tt.contextFunc))
			ctx := context.WithValue(context.Background(), "key", "original")
			var receivedCtx context.Context

			streamer := func(c context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
				receivedCtx = c
				if tt.streamerErr != nil {
					return nil, tt.streamerErr
				}
				return &mockClientStream{}, nil
			}

			stream, err := interceptor(ctx, nil, nil, "/test/method", streamer)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
				assert.Nil(t, stream)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, stream)
			}
			assert.Equal(t, tt.wantValue, receivedCtx.Value(tt.wantKey))
		})
	}
}

func TestStreamClientInterceptor_ContextPropagation(t *testing.T) {
	interceptor := StreamClientInterceptor(WithContextFunc(func(ctx context.Context) context.Context {
		return context.WithValue(ctx, "propagated", "yes")
	}))

	ctx := context.WithValue(context.Background(), "original", "value")
	var receivedCtx context.Context

	streamer := func(c context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		receivedCtx = c
		return &mockClientStream{}, nil
	}

	stream, err := interceptor(ctx, nil, nil, "/test/method", streamer)

	assert.NoError(t, err)
	assert.NotNil(t, stream)
	assert.Equal(t, "value", receivedCtx.Value("original"))
	assert.Equal(t, "yes", receivedCtx.Value("propagated"))
}

func TestMultipleOptions(t *testing.T) {
	interceptor := UnaryServerInterceptor(
		WithContextFunc(func(ctx context.Context) context.Context {
			return context.WithValue(ctx, "first", "1")
		}),
		WithContextFunc(func(ctx context.Context) context.Context {
			return context.WithValue(ctx, "second", "2")
		}),
	)

	var receivedCtx context.Context
	handler := func(c context.Context, req interface{}) (interface{}, error) {
		receivedCtx = c
		return nil, nil
	}

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)

	assert.NoError(t, err)
	assert.Equal(t, "2", receivedCtx.Value("second"))
}

func TestNilContextFunc(t *testing.T) {
	interceptor := UnaryServerInterceptor()
	
	ctx := context.WithValue(context.Background(), "key", "value")
	var receivedCtx context.Context
	
	handler := func(c context.Context, req interface{}) (interface{}, error) {
		receivedCtx = c
		return "resp", nil
	}
	
	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)
	
	assert.NoError(t, err)
	assert.Equal(t, "resp", resp)
	assert.Equal(t, "value", receivedCtx.Value("key"))
}
