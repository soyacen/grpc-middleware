package accesslog

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func mockUnaryHandler(result any, err error) grpc.UnaryHandler {
	return func(ctx context.Context, req any) (any, error) {
		return result, err
	}
}

func mockStreamHandler(err error) grpc.StreamHandler {
	return func(srv any, stream grpc.ServerStream) error {
		return err
	}
}

func mockUnaryInvoker(err error) grpc.UnaryInvoker {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		return err
	}
}

func mockStreamer(clientStream grpc.ClientStream, err error) grpc.Streamer {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		return clientStream, err
	}
}

func TestUnaryServerInterceptor(t *testing.T) {
	tests := []struct {
		name      string
		opts      []Option
		handler   grpc.UnaryHandler
		wantResp  any
		wantErr   error
		wantCall  bool
	}{
		{
			name:      "正常请求不跳过",
			opts:      []Option{},
			handler:   mockUnaryHandler("ok", nil),
			wantResp:  "ok",
			wantErr:   nil,
			wantCall:  true,
		},
		{
			name:      "handler返回错误不跳过",
			opts:      []Option{},
			handler:   mockUnaryHandler(nil, errors.New("handler error")),
			wantResp:  nil,
			wantErr:   errors.New("handler error"),
			wantCall:  true,
		},
		{
			name:      "skip返回true时跳过日志",
			opts:      []Option{WithSkip(func(string, error) bool { return true })},
			handler:   mockUnaryHandler("ok", nil),
			wantResp:  "ok",
			wantErr:   nil,
			wantCall:  true,
		},
		{
			name:      "skip返回false时不跳过",
			opts:      []Option{WithSkip(func(string, error) bool { return false })},
			handler:   mockUnaryHandler("ok", nil),
			wantResp:  "ok",
			wantErr:   nil,
			wantCall:  true,
		},
		{
			name:      "自定义level",
			opts:      []Option{WithLevel(slog.LevelDebug)},
			handler:   mockUnaryHandler("ok", nil),
			wantResp:  "ok",
			wantErr:   nil,
			wantCall:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			interceptor := UnaryServerInterceptor(tt.opts...)
			info := &grpc.UnaryServerInfo{FullMethod: "/test/method"}
			wrappedHandler := func(ctx context.Context, req any) (any, error) {
				called = true
				return tt.handler(ctx, req)
			}

			resp, err := interceptor(context.Background(), nil, info, wrappedHandler)

			assert.Equal(t, tt.wantCall, called, "handler调用状态不匹配")
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantResp, resp)
		})
	}
}

func TestStreamServerInterceptor(t *testing.T) {
	tests := []struct {
		name     string
		opts     []Option
		handler  grpc.StreamHandler
		wantErr  error
		wantCall bool
	}{
		{
			name:     "正常请求不跳过",
			opts:     []Option{},
			handler:  mockStreamHandler(nil),
			wantErr:  nil,
			wantCall: true,
		},
		{
			name:     "handler返回错误不跳过",
			opts:     []Option{},
			handler:  mockStreamHandler(errors.New("stream error")),
			wantErr:  errors.New("stream error"),
			wantCall: true,
		},
		{
			name:     "skip返回true时跳过日志",
			opts:     []Option{WithSkip(func(string, error) bool { return true })},
			handler:  mockStreamHandler(nil),
			wantErr:  nil,
			wantCall: true,
		},
		{
			name:     "skip返回false时不跳过",
			opts:     []Option{WithSkip(func(string, error) bool { return false })},
			handler:  mockStreamHandler(nil),
			wantErr:  nil,
			wantCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			interceptor := StreamServerInterceptor(tt.opts...)
			info := &grpc.StreamServerInfo{FullMethod: "/test/method"}
			mockStream := &mockServerStream{ctx: context.Background()}
			wrappedHandler := func(srv any, stream grpc.ServerStream) error {
				called = true
				return tt.handler(srv, stream)
			}

			err := interceptor(nil, mockStream, info, wrappedHandler)

			assert.Equal(t, tt.wantCall, called, "handler调用状态不匹配")
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUnaryClientInterceptor(t *testing.T) {
	tests := []struct {
		name     string
		opts     []Option
		invoker  grpc.UnaryInvoker
		wantErr  error
		wantCall bool
	}{
		{
			name:     "正常请求不跳过",
			opts:     []Option{},
			invoker:  mockUnaryInvoker(nil),
			wantErr:  nil,
			wantCall: true,
		},
		{
			name:     "invoker返回错误不跳过",
			opts:     []Option{},
			invoker:  mockUnaryInvoker(errors.New("invoker error")),
			wantErr:  errors.New("invoker error"),
			wantCall: true,
		},
		{
			name:     "skip返回true时跳过日志",
			opts:     []Option{WithSkip(func(string, error) bool { return true })},
			invoker:  mockUnaryInvoker(nil),
			wantErr:  nil,
			wantCall: true,
		},
		{
			name:     "skip返回false时不跳过",
			opts:     []Option{WithSkip(func(string, error) bool { return false })},
			invoker:  mockUnaryInvoker(nil),
			wantErr:  nil,
			wantCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			interceptor := UnaryClientInterceptor(tt.opts...)
			wrappedInvoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				called = true
				return tt.invoker(ctx, method, req, reply, cc, opts...)
			}

			err := interceptor(context.Background(), "/test/method", nil, nil, nil, wrappedInvoker)

			assert.Equal(t, tt.wantCall, called, "invoker调用状态不匹配")
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStreamClientInterceptor(t *testing.T) {
	tests := []struct {
		name      string
		opts      []Option
		streamer  grpc.Streamer
		wantErr   error
		wantCall  bool
	}{
		{
			name:      "正常请求不跳过",
			opts:      []Option{},
			streamer:  mockStreamer(nil, nil),
			wantErr:   nil,
			wantCall:  true,
		},
		{
			name:      "streamer返回错误不跳过",
			opts:      []Option{},
			streamer:  mockStreamer(nil, errors.New("streamer error")),
			wantErr:   errors.New("streamer error"),
			wantCall:  true,
		},
		{
			name:      "skip返回true时跳过日志",
			opts:      []Option{WithSkip(func(string, error) bool { return true })},
			streamer:  mockStreamer(nil, nil),
			wantErr:   nil,
			wantCall:  true,
		},
		{
			name:      "skip返回false时不跳过",
			opts:      []Option{WithSkip(func(string, error) bool { return false })},
			streamer:  mockStreamer(nil, nil),
			wantErr:   nil,
			wantCall:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			interceptor := StreamClientInterceptor(tt.opts...)
			wrappedStreamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
				called = true
				return tt.streamer(ctx, desc, cc, method, opts...)
			}

			_, err := interceptor(context.Background(), nil, nil, "/test/method", wrappedStreamer)

			assert.Equal(t, tt.wantCall, called, "streamer调用状态不匹配")
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type mockServerStream struct {
	ctx context.Context
}

func (m *mockServerStream) SetHeader(md metadata.MD) error  { return nil }
func (m *mockServerStream) SendHeader(md metadata.MD) error { return nil }
func (m *mockServerStream) SetTrailer(md metadata.MD)       {}
func (m *mockServerStream) Context() context.Context        { return m.ctx }
func (m *mockServerStream) SendMsg(msg any) error           { return nil }
func (m *mockServerStream) RecvMsg(msg any) error           { return nil }
