package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type mockServerStream struct {
	ctx       context.Context
	sendErr   error
	recvErr   error
	sendCount int
	recvCount int
}

func (m *mockServerStream) SetHeader(md metadata.MD) error  { return nil }
func (m *mockServerStream) SendHeader(md metadata.MD) error { return nil }
func (m *mockServerStream) SetTrailer(md metadata.MD)       {}
func (m *mockServerStream) Context() context.Context        { return m.ctx }
func (m *mockServerStream) SendMsg(v any) error {
	m.sendCount++
	return m.sendErr
}
func (m *mockServerStream) RecvMsg(v any) error {
	m.recvCount++
	return m.recvErr
}

func TestUnaryServerInterceptor_Success(t *testing.T) {
	tests := []struct {
		name         string
		authFunc     AuthFunc
		req          any
		handlerResp  any
		handlerErr   error
		wantResp     any
		wantErr      error
		wantAuthCall bool
	}{
		{
			name: "认证成功并继续处理请求",
			authFunc: func(ctx context.Context, fullMethod string) (context.Context, error) {
				newCtx := context.WithValue(ctx, "auth", "success")
				return newCtx, nil
			},
			req:          "request",
			handlerResp:  "response",
			handlerErr:   nil,
			wantResp:     "response",
			wantErr:      nil,
			wantAuthCall: true,
		},
		{
			name: "认证成功但handler返回错误",
			authFunc: func(ctx context.Context, fullMethod string) (context.Context, error) {
				return ctx, nil
			},
			req:          "request",
			handlerResp:  nil,
			handlerErr:   errors.New("handler error"),
			wantResp:     nil,
			wantErr:      errors.New("handler error"),
			wantAuthCall: true,
		},
		{
			name: "认证失败返回错误",
			authFunc: func(ctx context.Context, fullMethod string) (context.Context, error) {
				return nil, errors.New("auth failed")
			},
			req:          "request",
			handlerResp:  nil,
			handlerErr:   nil,
			wantResp:     nil,
			wantErr:      errors.New("auth failed"),
			wantAuthCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authCalled := false
			wrappedAuthFunc := func(ctx context.Context, fullMethod string) (context.Context, error) {
				authCalled = true
				return tt.authFunc(ctx, fullMethod)
			}

			handler := grpc.UnaryHandler(func(ctx context.Context, req any) (any, error) {
				return tt.handlerResp, tt.handlerErr
			})

			interceptor := UnaryServerInterceptor(wrappedAuthFunc)
			resp, err := interceptor(context.Background(), tt.req, &grpc.UnaryServerInfo{FullMethod: "/test/method"}, handler)

			assert.Equal(t, tt.wantAuthCall, authCalled)
			assert.Equal(t, tt.wantResp, resp)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUnaryServerInterceptor_ContextPropagation(t *testing.T) {
	tests := []struct {
		name             string
		authFunc         AuthFunc
		wantCtxKey       string
		wantCtxValue     any
		wantHandlerCtx   bool
	}{
		{
			name: "认证函数修改上下文并传递给handler",
			authFunc: func(ctx context.Context, fullMethod string) (context.Context, error) {
				return context.WithValue(ctx, "user_id", "12345"), nil
			},
			wantCtxKey:     "user_id",
			wantCtxValue:   "12345",
			wantHandlerCtx: true,
		},
		{
			name: "认证函数传递原始上下文",
			authFunc: func(ctx context.Context, fullMethod string) (context.Context, error) {
				return ctx, nil
			},
			wantCtxKey:     "original_key",
			wantCtxValue:   "original_value",
			wantHandlerCtx: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handlerCtx context.Context
			handler := grpc.UnaryHandler(func(ctx context.Context, req any) (any, error) {
				handlerCtx = ctx
				return "resp", nil
			})

			interceptor := UnaryServerInterceptor(tt.authFunc)
			ctx := context.WithValue(context.Background(), tt.wantCtxKey, tt.wantCtxValue)
			_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test/method"}, handler)

			assert.NoError(t, err)
			assert.NotNil(t, handlerCtx)
			if tt.wantHandlerCtx {
				assert.Equal(t, tt.wantCtxValue, handlerCtx.Value(tt.wantCtxKey))
			}
		})
	}
}

func TestUnaryServerInterceptor_FullMethodPassed(t *testing.T) {
	var receivedMethod string
	authFunc := func(ctx context.Context, fullMethod string) (context.Context, error) {
		receivedMethod = fullMethod
		return ctx, nil
	}

	handler := grpc.UnaryHandler(func(ctx context.Context, req any) (any, error) {
		return "resp", nil
	})

	interceptor := UnaryServerInterceptor(authFunc)
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/service/method"}, handler)

	assert.NoError(t, err)
	assert.Equal(t, "/service/method", receivedMethod)
}

func TestStreamServerInterceptor_Success(t *testing.T) {
	tests := []struct {
		name         string
		authFunc     AuthFunc
		handlerErr   error
		wantErr      error
		wantAuthCall bool
	}{
		{
			name: "认证成功并继续处理流请求",
			authFunc: func(ctx context.Context, fullMethod string) (context.Context, error) {
				return ctx, nil
			},
			handlerErr:   nil,
			wantErr:      nil,
			wantAuthCall: true,
		},
		{
			name: "认证成功但handler返回错误",
			authFunc: func(ctx context.Context, fullMethod string) (context.Context, error) {
				return ctx, nil
			},
			handlerErr:   errors.New("stream handler error"),
			wantErr:      errors.New("stream handler error"),
			wantAuthCall: true,
		},
		{
			name: "认证失败返回错误",
			authFunc: func(ctx context.Context, fullMethod string) (context.Context, error) {
				return nil, errors.New("stream auth failed")
			},
			handlerErr:   nil,
			wantErr:      errors.New("stream auth failed"),
			wantAuthCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authCalled := false
			wrappedAuthFunc := func(ctx context.Context, fullMethod string) (context.Context, error) {
				authCalled = true
				return tt.authFunc(ctx, fullMethod)
			}

			handler := grpc.StreamHandler(func(srv any, stream grpc.ServerStream) error {
				return tt.handlerErr
			})

			stream := &mockServerStream{ctx: context.Background()}
			interceptor := StreamServerInterceptor(wrappedAuthFunc)
			err := interceptor(nil, stream, &grpc.StreamServerInfo{FullMethod: "/test/stream"}, handler)

			assert.Equal(t, tt.wantAuthCall, authCalled)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStreamServerInterceptor_ContextPropagation(t *testing.T) {
	tests := []struct {
		name         string
		authFunc     AuthFunc
		wantCtxValue any
	}{
		{
			name: "认证函数修改流上下文",
			authFunc: func(ctx context.Context, fullMethod string) (context.Context, error) {
				return context.WithValue(ctx, "stream_user", "user123"), nil
			},
			wantCtxValue: "user123",
		},
		{
			name: "认证函数传递带原始值的上下文",
			authFunc: func(ctx context.Context, fullMethod string) (context.Context, error) {
				return ctx, nil
			},
			wantCtxValue: "original_value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handlerStream grpc.ServerStream
			handler := grpc.StreamHandler(func(srv any, stream grpc.ServerStream) error {
				handlerStream = stream
				return nil
			})

			interceptor := StreamServerInterceptor(tt.authFunc)
			ctx := context.WithValue(context.Background(), "stream_user", tt.wantCtxValue)
			stream := &mockServerStream{ctx: ctx}
			err := interceptor(nil, stream, &grpc.StreamServerInfo{FullMethod: "/test/stream"}, handler)

			assert.NoError(t, err)
			assert.NotNil(t, handlerStream)
			assert.Equal(t, tt.wantCtxValue, handlerStream.Context().Value("stream_user"))
		})
	}
}

func TestStreamServerInterceptor_WrapsStream(t *testing.T) {
	authFunc := func(ctx context.Context, fullMethod string) (context.Context, error) {
		newCtx := context.WithValue(ctx, "wrapped", true)
		return newCtx, nil
	}

	var receivedStream grpc.ServerStream
	handler := grpc.StreamHandler(func(srv any, stream grpc.ServerStream) error {
		receivedStream = stream
		return nil
	})

	interceptor := StreamServerInterceptor(authFunc)
	stream := &mockServerStream{ctx: context.Background()}
	err := interceptor(nil, stream, &grpc.StreamServerInfo{FullMethod: "/test/stream"}, handler)

	assert.NoError(t, err)
	assert.NotNil(t, receivedStream)

		wrapped, ok := receivedStream.(*WrappedServerStream)
	assert.True(t, ok, "stream should be wrapped")
	assert.NotNil(t, wrapped)
	assert.Equal(t, true, wrapped.Context().Value("wrapped"))
}

func TestStreamServerInterceptor_FullMethodPassed(t *testing.T) {
	var receivedMethod string
	authFunc := func(ctx context.Context, fullMethod string) (context.Context, error) {
		receivedMethod = fullMethod
		return ctx, nil
	}

	handler := grpc.StreamHandler(func(srv any, stream grpc.ServerStream) error {
		return nil
	})

	interceptor := StreamServerInterceptor(authFunc)
	stream := &mockServerStream{ctx: context.Background()}
	_ = interceptor(nil, stream, &grpc.StreamServerInfo{FullMethod: "/service/stream"}, handler)

	assert.Equal(t, "/service/stream", receivedMethod)
}

func TestWrapServerStream(t *testing.T) {
	tests := []struct {
		name        string
		stream      grpc.ServerStream
		wantSame    bool
		wantWrapped bool
	}{
		{
			name:        "包装普通ServerStream",
			stream:      &mockServerStream{ctx: context.Background()},
			wantSame:    false,
			wantWrapped: true,
		},
		{
			name:        "已经是WrappedServerStream则直接返回",
			stream:      &WrappedServerStream{ServerStream: &mockServerStream{ctx: context.Background()}, WrappedContext: context.Background()},
			wantSame:    true,
			wantWrapped: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := WrapServerStream(tt.stream)

			assert.NotNil(t, wrapped)
			if tt.wantSame {
				assert.Equal(t, tt.stream, wrapped, "应该返回同一个实例")
			} else {
				assert.NotEqual(t, tt.stream, wrapped, "应该创建新的包装实例")
			}

			assert.True(t, wrapped != nil)
		})
	}
}

func TestWrappedServerStream_Context(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		wantVal any
	}{
		{
			name:    "返回包装后的上下文",
			ctx:     context.WithValue(context.Background(), "key", "value"),
			wantVal: "value",
		},
		{
			name:    "返回空上下文",
			ctx:     context.Background(),
			wantVal: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream := &WrappedServerStream{
				ServerStream:   &mockServerStream{ctx: context.Background()},
				WrappedContext: tt.ctx,
			}

			ctx := stream.Context()
			assert.Equal(t, tt.ctx, ctx)
			assert.Equal(t, tt.wantVal, ctx.Value("key"))
		})
	}
}

func TestWrappedServerStream_Methods(t *testing.T) {
	mock := &mockServerStream{ctx: context.Background()}
	wrapped := WrapServerStream(mock)

	err := wrapped.SendMsg("test")
	assert.NoError(t, err)
	assert.Equal(t, 1, mock.sendCount)

	err = wrapped.RecvMsg("test")
	assert.NoError(t, err)
	assert.Equal(t, 1, mock.recvCount)
}

func TestWrappedServerStream_NestedWrap(t *testing.T) {
	mock := &mockServerStream{ctx: context.Background()}
	wrapped1 := WrapServerStream(mock)
	wrapped2 := WrapServerStream(wrapped1)

	assert.Equal(t, wrapped1, wrapped2, "多次包装应该返回同一个实例")
}
