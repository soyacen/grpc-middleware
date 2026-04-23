package ratelimiter

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type testMockRateLimiter struct {
	allowFunc func() (func(DoneInfo), error)
}

func (m *testMockRateLimiter) Allow() (func(DoneInfo), error) {
	return m.allowFunc()
}

type mockUnaryHandler struct {
	resp      interface{}
	err       error
	panicVal  interface{}
	callCount int
}

func (m *mockUnaryHandler) handle(ctx context.Context, req interface{}) (interface{}, error) {
	m.callCount++
	if m.panicVal != nil {
		panic(m.panicVal)
	}
	return m.resp, m.err
}

type mockStreamHandler struct {
	err       error
	panicVal  interface{}
	callCount int
}

func (m *mockStreamHandler) handle(srv interface{}, stream grpc.ServerStream) error {
	m.callCount++
	if m.panicVal != nil {
		panic(m.panicVal)
	}
	return m.err
}

type mockServerStream struct{}

func (m *mockServerStream) SetHeader(md metadata.MD) error  { return nil }
func (m *mockServerStream) SendHeader(md metadata.MD) error { return nil }
func (m *mockServerStream) SetTrailer(md metadata.MD)       {}
func (m *mockServerStream) Context() context.Context        { return context.Background() }
func (m *mockServerStream) SendMsg(v interface{}) error     { return nil }
func (m *mockServerStream) RecvMsg(v interface{}) error     { return nil }

func TestUnaryServerInterceptor_Success(t *testing.T) {
	tests := []struct {
		name       string
		resp       interface{}
		handlerErr error
		wantResp   interface{}
		wantErr    error
	}{
		{"success_with_response", "hello", nil, "hello", nil},
		{"success_with_nil_response", nil, nil, nil, nil},
		{"handler_returns_error", nil, errors.New("handler error"), nil, errors.New("handler error")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doneCalled := false
			var doneInfo DoneInfo
			limiter := &testMockRateLimiter{
				allowFunc: func() (func(DoneInfo), error) {
					return func(di DoneInfo) {
						doneCalled = true
						doneInfo = di
					}, nil
				},
			}

			handler := &mockUnaryHandler{resp: tt.resp, err: tt.handlerErr}
			interceptor := UnaryServerInterceptor(withRateLimiter(limiter))
			resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler.handle)

			assert.Equal(t, tt.wantResp, resp)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.True(t, doneCalled)
			assert.Equal(t, tt.handlerErr, doneInfo.Err)
			assert.Equal(t, 1, handler.callCount)
		})
	}
}

func TestUnaryServerInterceptor_SkipTrue(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"skip_returns_true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := &testMockRateLimiter{
				allowFunc: func() (func(DoneInfo), error) {
					t.Fatal("Allow should not be called when Skip returns true")
					return nil, nil
				},
			}

			handler := &mockUnaryHandler{resp: "resp"}
			interceptor := UnaryServerInterceptor(
				withRateLimiter(limiter),
				WithSkip(func() bool { return true }),
			)
			resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler.handle)

			assert.NoError(t, err)
			assert.Equal(t, "resp", resp)
			assert.Equal(t, 1, handler.callCount)
		})
	}
}

func TestUnaryServerInterceptor_SkipFalse(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"skip_returns_false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowCalled := false
			limiter := &testMockRateLimiter{
				allowFunc: func() (func(DoneInfo), error) {
					allowCalled = true
					return func(DoneInfo) {}, nil
				},
			}

			handler := &mockUnaryHandler{resp: "resp"}
			interceptor := UnaryServerInterceptor(
				withRateLimiter(limiter),
				WithSkip(func() bool { return false }),
			)
			resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler.handle)

			assert.NoError(t, err)
			assert.Equal(t, "resp", resp)
			assert.True(t, allowCalled)
			assert.Equal(t, 1, handler.callCount)
		})
	}
}

func TestUnaryServerInterceptor_LimitExceeded(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"limit_exceeded"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := &testMockRateLimiter{
				allowFunc: func() (func(DoneInfo), error) {
					return nil, ErrLimitExceeded
				},
			}

			handler := &mockUnaryHandler{resp: "resp"}
			interceptor := UnaryServerInterceptor(withRateLimiter(limiter))
			resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler.handle)

			assert.Equal(t, ErrLimitExceeded, err)
			assert.Nil(t, resp)
			assert.Equal(t, 0, handler.callCount)
		})
	}
}

func TestUnaryServerInterceptor_PanicRecovery(t *testing.T) {
	tests := []struct {
		name     string
		panicVal interface{}
	}{
		{"panic_with_string", "something went wrong"},
		{"panic_with_error", errors.New("panic error")},
		{"panic_with_int", 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doneCalled := false
			var doneInfo DoneInfo
			limiter := &testMockRateLimiter{
				allowFunc: func() (func(DoneInfo), error) {
					return func(di DoneInfo) {
						doneCalled = true
						doneInfo = di
					}, nil
				},
			}

			handler := &mockUnaryHandler{panicVal: tt.panicVal}
			interceptor := UnaryServerInterceptor(withRateLimiter(limiter))

			assert.Panics(t, func() {
				interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler.handle)
			})

			assert.True(t, doneCalled)
			assert.Error(t, doneInfo.Err)
			assert.Contains(t, doneInfo.Err.Error(), "panic:")
		})
	}
}

func TestUnaryServerInterceptor_PanicRepanic(t *testing.T) {
	tests := []struct {
		name      string
		panicVal  interface{}
		wantPanic interface{}
	}{
		{"repanic_string", "critical error", "critical error"},
		{"repanic_error", errors.New("boom"), errors.New("boom")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := &testMockRateLimiter{
				allowFunc: func() (func(DoneInfo), error) {
					return func(DoneInfo) {}, nil
				},
			}

			handler := &mockUnaryHandler{panicVal: tt.panicVal}
			interceptor := UnaryServerInterceptor(withRateLimiter(limiter))

			defer func() {
				r := recover()
				assert.NotNil(t, r)
				assert.Equal(t, tt.wantPanic, r)
			}()

			interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler.handle)
			t.Fatal("expected panic")
		})
	}
}

func TestUnaryServerInterceptor_ErrorCallback(t *testing.T) {
	tests := []struct {
		name       string
		handlerErr error
		wantErr    error
	}{
		{"handler_error", errors.New("handler failed"), errors.New("handler failed")},
		{"handler_no_error", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doneCalled := false
			var doneInfo DoneInfo
			limiter := &testMockRateLimiter{
				allowFunc: func() (func(DoneInfo), error) {
					return func(di DoneInfo) {
						doneCalled = true
						doneInfo = di
					}, nil
				},
			}

			handler := &mockUnaryHandler{resp: "resp", err: tt.handlerErr}
			interceptor := UnaryServerInterceptor(withRateLimiter(limiter))
			_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler.handle)

			assert.True(t, doneCalled)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
				assert.EqualError(t, doneInfo.Err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.NoError(t, doneInfo.Err)
			}
		})
	}
}

func TestUnaryServerInterceptor_ContextPropagation(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"context_propagated_to_handler"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), "key", "value")
			var receivedCtx context.Context

			limiter := &testMockRateLimiter{
				allowFunc: func() (func(DoneInfo), error) {
					return func(DoneInfo) {}, nil
				},
			}

			handler := grpc.UnaryHandler(func(c context.Context, req interface{}) (interface{}, error) {
				receivedCtx = c
				return "resp", nil
			})
			interceptor := UnaryServerInterceptor(withRateLimiter(limiter))
			resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)

			assert.NoError(t, err)
			assert.Equal(t, "resp", resp)
			assert.Equal(t, "value", receivedCtx.Value("key"))
		})
	}
}

func TestStreamServerInterceptor_Success(t *testing.T) {
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
			doneCalled := false
			var doneInfo DoneInfo
			limiter := &testMockRateLimiter{
				allowFunc: func() (func(DoneInfo), error) {
					return func(di DoneInfo) {
						doneCalled = true
						doneInfo = di
					}, nil
				},
			}

			handler := &mockStreamHandler{err: tt.handlerErr}
			interceptor := StreamServerInterceptor(withRateLimiter(limiter))
			stream := &mockServerStream{}
			err := interceptor(nil, stream, &grpc.StreamServerInfo{}, handler.handle)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.True(t, doneCalled)
			assert.Equal(t, tt.handlerErr, doneInfo.Err)
			assert.Equal(t, 1, handler.callCount)
		})
	}
}

func TestStreamServerInterceptor_SkipTrue(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"skip_returns_true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := &testMockRateLimiter{
				allowFunc: func() (func(DoneInfo), error) {
					t.Fatal("Allow should not be called when Skip returns true")
					return nil, nil
				},
			}

			handler := &mockStreamHandler{}
			interceptor := StreamServerInterceptor(
				withRateLimiter(limiter),
				WithSkip(func() bool { return true }),
			)
			stream := &mockServerStream{}
			err := interceptor(nil, stream, &grpc.StreamServerInfo{}, handler.handle)

			assert.NoError(t, err)
			assert.Equal(t, 1, handler.callCount)
		})
	}
}

func TestStreamServerInterceptor_SkipFalse(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"skip_returns_false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowCalled := false
			limiter := &testMockRateLimiter{
				allowFunc: func() (func(DoneInfo), error) {
					allowCalled = true
					return func(DoneInfo) {}, nil
				},
			}

			handler := &mockStreamHandler{}
			interceptor := StreamServerInterceptor(
				withRateLimiter(limiter),
				WithSkip(func() bool { return false }),
			)
			stream := &mockServerStream{}
			err := interceptor(nil, stream, &grpc.StreamServerInfo{}, handler.handle)

			assert.NoError(t, err)
			assert.True(t, allowCalled)
			assert.Equal(t, 1, handler.callCount)
		})
	}
}

func TestStreamServerInterceptor_LimitExceeded(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"limit_exceeded"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := &testMockRateLimiter{
				allowFunc: func() (func(DoneInfo), error) {
					return nil, ErrLimitExceeded
				},
			}

			handler := &mockStreamHandler{}
			interceptor := StreamServerInterceptor(withRateLimiter(limiter))
			stream := &mockServerStream{}
			err := interceptor(nil, stream, &grpc.StreamServerInfo{}, handler.handle)

			assert.Equal(t, ErrLimitExceeded, err)
			assert.Equal(t, 0, handler.callCount)
		})
	}
}

func TestStreamServerInterceptor_PanicRecovery(t *testing.T) {
	tests := []struct {
		name     string
		panicVal interface{}
	}{
		{"panic_with_string", "stream panic"},
		{"panic_with_error", errors.New("stream panic error")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doneCalled := false
			var doneInfo DoneInfo
			limiter := &testMockRateLimiter{
				allowFunc: func() (func(DoneInfo), error) {
					return func(di DoneInfo) {
						doneCalled = true
						doneInfo = di
					}, nil
				},
			}

			handler := &mockStreamHandler{panicVal: tt.panicVal}
			interceptor := StreamServerInterceptor(withRateLimiter(limiter))

			assert.Panics(t, func() {
				interceptor(nil, nil, &grpc.StreamServerInfo{}, handler.handle)
			})

			assert.True(t, doneCalled)
			assert.Error(t, doneInfo.Err)
			assert.Contains(t, doneInfo.Err.Error(), "panic:")
		})
	}
}

func TestStreamServerInterceptor_PanicRepanic(t *testing.T) {
	tests := []struct {
		name      string
		panicVal  interface{}
		wantPanic interface{}
	}{
		{"repanic_string", "stream critical", "stream critical"},
		{"repanic_error", errors.New("stream boom"), errors.New("stream boom")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := &testMockRateLimiter{
				allowFunc: func() (func(DoneInfo), error) {
					return func(DoneInfo) {}, nil
				},
			}

			handler := &mockStreamHandler{panicVal: tt.panicVal}
			interceptor := StreamServerInterceptor(withRateLimiter(limiter))

			defer func() {
				r := recover()
				assert.NotNil(t, r)
				assert.Equal(t, tt.wantPanic, r)
			}()

			interceptor(nil, nil, &grpc.StreamServerInfo{}, handler.handle)
			t.Fatal("expected panic")
		})
	}
}

func TestStreamServerInterceptor_ErrorCallback(t *testing.T) {
	tests := []struct {
		name       string
		handlerErr error
		wantErr    error
	}{
		{"handler_error", errors.New("stream handler failed"), errors.New("stream handler failed")},
		{"handler_no_error", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doneCalled := false
			var doneInfo DoneInfo
			limiter := &testMockRateLimiter{
				allowFunc: func() (func(DoneInfo), error) {
					return func(di DoneInfo) {
						doneCalled = true
						doneInfo = di
					}, nil
				},
			}

			handler := &mockStreamHandler{err: tt.handlerErr}
			interceptor := StreamServerInterceptor(withRateLimiter(limiter))
			stream := &mockServerStream{}
			err := interceptor(nil, stream, &grpc.StreamServerInfo{}, handler.handle)

			assert.True(t, doneCalled)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
				assert.EqualError(t, doneInfo.Err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.NoError(t, doneInfo.Err)
			}
		})
	}
}
