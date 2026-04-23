package timeout

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

type mockInvoker struct {
	err        error
	callCount  int
	lastCtx    context.Context
	lastMethod string
	lastReq    interface{}
	lastReply  interface{}
	lastCC     *grpc.ClientConn
	delay      time.Duration
}

func (m *mockInvoker) invoke(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
	m.callCount++
	m.lastCtx = ctx
	m.lastMethod = method
	m.lastReq = req
	m.lastReply = reply
	m.lastCC = cc
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return m.err
}

func TestUnaryClientInterceptor_DefaultTimeout(t *testing.T) {
	interceptor := UnaryClientInterceptor()
	mock := &mockInvoker{err: nil}

	start := time.Now()
	err := interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, 1, mock.callCount)

	deadline, ok := mock.lastCtx.Deadline()
	assert.True(t, ok, "context should have deadline")
	assert.WithinDuration(t, start.Add(5*time.Second), deadline, time.Second, "default timeout should be 5 seconds")
	assert.Less(t, elapsed, 100*time.Millisecond, "should complete quickly")
}

func TestUnaryClientInterceptor_CustomTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{"timeout_1_second", time.Second},
		{"timeout_3_seconds", 3 * time.Second},
		{"timeout_100_milliseconds", 100 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := UnaryClientInterceptor(Timeout(tt.timeout))
			mock := &mockInvoker{err: nil}

			start := time.Now()
			err := interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke)
			elapsed := time.Since(start)

			assert.NoError(t, err)
			assert.Equal(t, 1, mock.callCount)

			deadline, ok := mock.lastCtx.Deadline()
			assert.True(t, ok, "context should have deadline")
			assert.WithinDuration(t, start.Add(tt.timeout), deadline, 100*time.Millisecond)
			assert.Less(t, elapsed, 50*time.Millisecond, "should complete quickly")
		})
	}
}

func TestUnaryClientInterceptor_NormalRequest(t *testing.T) {
	interceptor := UnaryClientInterceptor(Timeout(time.Second))
	mock := &mockInvoker{err: nil}

	err := interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke)

	assert.NoError(t, err)
	assert.Equal(t, 1, mock.callCount)
}

func TestUnaryClientInterceptor_TimeoutTriggered(t *testing.T) {
	interceptor := UnaryClientInterceptor(Timeout(50 * time.Millisecond))
	mock := &mockInvoker{delay: 200 * time.Millisecond, err: nil}

	start := time.Now()
	err := interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke)
	elapsed := time.Since(start)

	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Equal(t, 1, mock.callCount)
	assert.Less(t, elapsed, 150*time.Millisecond, "should return around timeout duration")
	assert.GreaterOrEqual(t, elapsed, 40*time.Millisecond)
}

func TestUnaryClientInterceptor_ErrorPropagation(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"custom_error", errors.New("custom error")},
		{"another_error", errors.New("another error")},
		{"nil_error", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := UnaryClientInterceptor(Timeout(time.Second))
			mock := &mockInvoker{err: tt.err}

			err := interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke)

			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, 1, mock.callCount)
		})
	}
}

func TestUnaryClientInterceptor_ContextPropagation(t *testing.T) {
	interceptor := UnaryClientInterceptor(Timeout(time.Second))
	mock := &mockInvoker{err: nil}

	ctx := context.WithValue(context.Background(), "key", "value")
	err := interceptor(ctx, "/test/method", nil, nil, nil, mock.invoke)

	assert.NoError(t, err)
	assert.Equal(t, "value", mock.lastCtx.Value("key"), "context value should be propagated")

	_, ok := mock.lastCtx.Deadline()
	assert.True(t, ok, "context should have deadline")
}

func TestUnaryClientInterceptor_ParamsPropagation(t *testing.T) {
	interceptor := UnaryClientInterceptor(Timeout(time.Second))
	mock := &mockInvoker{err: nil}

	req := "request"
	reply := "reply"
	method := "/test.Service/Method"

	err := interceptor(context.Background(), method, req, &reply, nil, mock.invoke)

	assert.NoError(t, err)
	assert.Equal(t, method, mock.lastMethod)
	assert.Equal(t, req, mock.lastReq)
	assert.Equal(t, &reply, mock.lastReply)
	assert.Nil(t, mock.lastCC)
}

func TestUnaryClientInterceptor_MultipleMethods(t *testing.T) {
	interceptor := UnaryClientInterceptor(Timeout(time.Second))

	methods := []string{
		"/service1/method1",
		"/service1/method2",
		"/service2/method1",
	}

	for _, method := range methods {
		mock := &mockInvoker{err: nil}
		err := interceptor(context.Background(), method, nil, nil, nil, mock.invoke)
		assert.NoError(t, err)
		assert.Equal(t, method, mock.lastMethod)
	}
}

func TestUnaryClientInterceptor_ContextCancelled(t *testing.T) {
	interceptor := UnaryClientInterceptor(Timeout(time.Hour))
	mock := &mockInvoker{delay: 200 * time.Millisecond, err: nil}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	err := interceptor(ctx, "/test/method", nil, nil, nil, mock.invoke)
	elapsed := time.Since(start)

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.Less(t, elapsed, 150*time.Millisecond)
	assert.GreaterOrEqual(t, elapsed, 30*time.Millisecond)
}

func TestUnaryClientInterceptor_ZeroTimeout(t *testing.T) {
	interceptor := UnaryClientInterceptor(Timeout(0))
	mock := &mockInvoker{err: nil}

	start := time.Now()
	err := interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, 1, mock.callCount)
	assert.Less(t, elapsed, 10*time.Millisecond)
}

func TestUnaryClientInterceptor_CallOptionsPassed(t *testing.T) {
	interceptor := UnaryClientInterceptor(Timeout(time.Second))

	mock := &mockInvoker{err: nil}
	err := interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke, grpc.FailFast(false))

	assert.NoError(t, err)
	assert.Equal(t, 1, mock.callCount)
}
