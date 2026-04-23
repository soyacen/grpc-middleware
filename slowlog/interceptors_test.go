package slowlog

import (
	"context"
	"errors"
	"testing"
	"time"

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
		threshold time.Duration
		sleepTime time.Duration
		handler   grpc.UnaryHandler
		wantErr   error
	}{
		{
			name:      "正常请求未触发慢日志",
			threshold: time.Hour,
			sleepTime: 0,
			handler:   mockUnaryHandler("ok", nil),
			wantErr:   nil,
		},
		{
			name:      "慢请求触发日志但handler正常返回",
			threshold: time.Nanosecond,
			sleepTime: time.Millisecond,
			handler:   mockUnaryHandler("ok", nil),
			wantErr:   nil,
		},
		{
			name:      "handler返回错误",
			threshold: time.Hour,
			sleepTime: 0,
			handler:   mockUnaryHandler(nil, errors.New("handler error")),
			wantErr:   errors.New("handler error"),
		},
		{
			name:      "边界值等于阈值不触发",
			threshold: time.Millisecond * 10,
			sleepTime: time.Millisecond * 5,
			handler:   mockUnaryHandler("ok", nil),
			wantErr:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := UnaryServerInterceptor(SlowRequestThreshold(tt.threshold))
			info := &grpc.UnaryServerInfo{FullMethod: "/test/method"}
			handler := func(ctx context.Context, req any) (any, error) {
				if tt.sleepTime > 0 {
					time.Sleep(tt.sleepTime)
				}
				return tt.handler(ctx, req)
			}

			resp, err := interceptor(context.Background(), nil, info, handler)

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.wantErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "ok", resp)
			}
		})
	}
}

func TestStreamServerInterceptor(t *testing.T) {
	tests := []struct {
		name      string
		threshold time.Duration
		sleepTime time.Duration
		handler   grpc.StreamHandler
		wantErr   error
	}{
		{
			name:      "正常请求未触发慢日志",
			threshold: time.Hour,
			sleepTime: 0,
			handler:   mockStreamHandler(nil),
			wantErr:   nil,
		},
		{
			name:      "慢请求触发日志",
			threshold: time.Nanosecond,
			sleepTime: time.Millisecond,
			handler:   mockStreamHandler(nil),
			wantErr:   nil,
		},
		{
			name:      "handler返回错误",
			threshold: time.Hour,
			sleepTime: 0,
			handler:   mockStreamHandler(errors.New("stream error")),
			wantErr:   errors.New("stream error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := StreamServerInterceptor(SlowRequestThreshold(tt.threshold))
			info := &grpc.StreamServerInfo{FullMethod: "/test/method"}

			mockStream := &mockServerStream{ctx: context.Background()}
			handler := func(srv any, stream grpc.ServerStream) error {
				if tt.sleepTime > 0 {
					time.Sleep(tt.sleepTime)
				}
				return tt.handler(srv, stream)
			}

			err := interceptor(nil, mockStream, info, handler)

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
		name      string
		threshold time.Duration
		sleepTime time.Duration
		invoker   grpc.UnaryInvoker
		wantErr   error
	}{
		{
			name:      "正常请求未触发慢日志",
			threshold: time.Hour,
			sleepTime: 0,
			invoker:   mockUnaryInvoker(nil),
			wantErr:   nil,
		},
		{
			name:      "慢请求触发日志",
			threshold: time.Nanosecond,
			sleepTime: time.Millisecond,
			invoker:   mockUnaryInvoker(nil),
			wantErr:   nil,
		},
		{
			name:      "invoker返回错误",
			threshold: time.Hour,
			sleepTime: 0,
			invoker:   mockUnaryInvoker(errors.New("invoker error")),
			wantErr:   errors.New("invoker error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := UnaryClientInterceptor(SlowRequestThreshold(tt.threshold))
			invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				if tt.sleepTime > 0 {
					time.Sleep(tt.sleepTime)
				}
				return tt.invoker(ctx, method, req, reply, cc, opts...)
			}

			err := interceptor(context.Background(), "/test/method", nil, nil, nil, invoker)

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
		threshold time.Duration
		sleepTime time.Duration
		streamer  grpc.Streamer
		wantErr   error
	}{
		{
			name:      "正常请求未触发慢日志",
			threshold: time.Hour,
			sleepTime: 0,
			streamer:  mockStreamer(nil, nil),
			wantErr:   nil,
		},
		{
			name:      "慢请求触发日志",
			threshold: time.Nanosecond,
			sleepTime: time.Millisecond,
			streamer:  mockStreamer(nil, nil),
			wantErr:   nil,
		},
		{
			name:      "streamer返回错误",
			threshold: time.Hour,
			sleepTime: 0,
			streamer:  mockStreamer(nil, errors.New("streamer error")),
			wantErr:   errors.New("streamer error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := StreamClientInterceptor(SlowRequestThreshold(tt.threshold))
			streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
				if tt.sleepTime > 0 {
					time.Sleep(tt.sleepTime)
				}
				return tt.streamer(ctx, desc, cc, method, opts...)
			}

			_, err := interceptor(context.Background(), nil, nil, "/test/method", streamer)

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
func (m *mockServerStream) Context() context.Context               { return m.ctx }
func (m *mockServerStream) SendMsg(msg any) error                  { return nil }
func (m *mockServerStream) RecvMsg(msg any) error                  { return nil }
