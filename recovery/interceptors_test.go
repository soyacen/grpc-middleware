package recovery

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type mockServerStream struct{}

func (m *mockServerStream) SetHeader(md metadata.MD) error  { return nil }
func (m *mockServerStream) SendHeader(md metadata.MD) error { return nil }
func (m *mockServerStream) SetTrailer(md metadata.MD)       {}
func (m *mockServerStream) Context() context.Context        { return context.Background() }
func (m *mockServerStream) SendMsg(v any) error             { return nil }
func (m *mockServerStream) RecvMsg(v any) error             { return nil }

func TestUnaryServerInterceptor_NoPanic(t *testing.T) {
	tests := []struct {
		name       string
		resp       any
		handlerErr error
		wantResp   any
		wantErr    error
	}{
		{"success_with_response", "hello", nil, "hello", nil},
		{"success_with_nil_response", nil, nil, nil, nil},
		{"handler_returns_error", nil, errors.New("handler error"), nil, errors.New("handler error")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := func(ctx context.Context, req any) (any, error) {
				return tt.resp, tt.handlerErr
			}

			interceptor := UnaryServerInterceptor()
			resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}, handler)

			assert.Equal(t, tt.wantResp, resp)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUnaryServerInterceptor_PanicRecovery(t *testing.T) {
	tests := []struct {
		name     string
		panicVal any
	}{
		{"panic_with_string", "something went wrong"},
		{"panic_with_error", errors.New("panic error")},
		{"panic_with_int", 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := func(ctx context.Context, req any) (any, error) {
				panic(tt.panicVal)
			}

			interceptor := UnaryServerInterceptor()
			resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}, handler)

			assert.Nil(t, resp)
			assert.Error(t, err)

			var panicErr *PanicError
			assert.ErrorAs(t, err, &panicErr)
			assert.Equal(t, "/test.Service/Method", panicErr.Method)
			assert.Equal(t, tt.panicVal, panicErr.Panic)
			assert.NotNil(t, panicErr.Stack)
		})
	}
}

func TestUnaryServerInterceptor_CustomHandler(t *testing.T) {
	customCalled := false
	customHandler := func(ctx context.Context, method string, p any) error {
		customCalled = true
		return errors.New("custom panic error: " + method)
	}

	handler := func(ctx context.Context, req any) (any, error) {
		panic("panic")
	}

	interceptor := UnaryServerInterceptor(RecoveryHandler(customHandler))
	resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}, handler)

	assert.Nil(t, resp)
	assert.EqualError(t, err, "custom panic error: /test.Service/Method")
	assert.True(t, customCalled)
}

func TestUnaryServerInterceptor_ContextPropagation(t *testing.T) {
	ctx := context.WithValue(context.Background(), "key", "value")
	var receivedCtx context.Context

	handler := func(ctx context.Context, req any) (any, error) {
		receivedCtx = ctx
		return "resp", nil
	}

	interceptor := UnaryServerInterceptor()
	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)

	assert.NoError(t, err)
	assert.Equal(t, "resp", resp)
	assert.Equal(t, "value", receivedCtx.Value("key"))
}

func TestStreamServerInterceptor_NoPanic(t *testing.T) {
	tests := []struct {
		name       string
		handlerErr error
		wantErr    error
	}{
		{"success_no_error", nil, nil},
		{"handler_returns_error", errors.New("stream error"), errors.New("stream error")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := func(srv any, stream grpc.ServerStream) error {
				return tt.handlerErr
			}

			interceptor := StreamServerInterceptor()
			stream := &mockServerStream{}
			err := interceptor(nil, stream, &grpc.StreamServerInfo{FullMethod: "/test.Service/StreamMethod"}, handler)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStreamServerInterceptor_PanicRecovery(t *testing.T) {
	tests := []struct {
		name     string
		panicVal any
	}{
		{"panic_with_string", "stream panic"},
		{"panic_with_error", errors.New("stream panic error")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := func(srv any, stream grpc.ServerStream) error {
				panic(tt.panicVal)
			}

			interceptor := StreamServerInterceptor()
			stream := &mockServerStream{}
			err := interceptor(nil, stream, &grpc.StreamServerInfo{FullMethod: "/test.Service/StreamMethod"}, handler)

			assert.Error(t, err)

			var panicErr *PanicError
			assert.ErrorAs(t, err, &panicErr)
			assert.Equal(t, "/test.Service/StreamMethod", panicErr.Method)
			assert.Equal(t, tt.panicVal, panicErr.Panic)
			assert.NotNil(t, panicErr.Stack)
		})
	}
}

func TestStreamServerInterceptor_CustomHandler(t *testing.T) {
	customCalled := false
	customHandler := func(ctx context.Context, method string, p any) error {
		customCalled = true
		return errors.New("custom stream panic: " + method)
	}

	handler := func(srv any, stream grpc.ServerStream) error {
		panic("stream panic")
	}

	interceptor := StreamServerInterceptor(RecoveryHandler(customHandler))
	stream := &mockServerStream{}
	err := interceptor(nil, stream, &grpc.StreamServerInfo{FullMethod: "/test.Service/StreamMethod"}, handler)

	assert.EqualError(t, err, "custom stream panic: /test.Service/StreamMethod")
	assert.True(t, customCalled)
}

func TestPanicError_Error(t *testing.T) {
	err := &PanicError{
		Method: "/test.Service/Method",
		Panic:  "something went wrong",
		Stack:  []byte("stack trace"),
	}

	want := "panic caught: /test.Service/Method\n\nsomething went wrong\n\nstack trace"
	assert.Equal(t, want, err.Error())
}
