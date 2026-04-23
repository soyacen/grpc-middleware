package retry

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockInvoker struct {
	errs       []error
	callCount  int
	lastCtx    context.Context
	lastMethod string
}

func (m *mockInvoker) invoke(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
	m.callCount++
	m.lastCtx = ctx
	m.lastMethod = method
	if m.callCount <= len(m.errs) {
		return m.errs[m.callCount-1]
	}
	return nil
}

type mockClientStream struct {
	grpc.ClientStream
	recvErr    error
	recvCalled int
}

func (m *mockClientStream) RecvMsg(msg interface{}) error {
	m.recvCalled++
	return m.recvErr
}

func (m *mockClientStream) CloseSend() error {
	return nil
}

type mockStreamer struct {
	errs         []error
	clientStream grpc.ClientStream
	callCount    int
	lastCtx      context.Context
	lastMethod   string
}

func (m *mockStreamer) stream(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	m.callCount++
	m.lastCtx = ctx
	m.lastMethod = method
	if m.callCount <= len(m.errs) {
		return nil, m.errs[m.callCount-1]
	}
	return m.clientStream, nil
}

func TestSleepWithContext_Normal(t *testing.T) {
	ctx := context.Background()
	start := time.Now()
	err := sleepWithContext(ctx, 10*time.Millisecond)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.True(t, elapsed >= 10*time.Millisecond, "should sleep at least 10ms")
}

func TestSleepWithContext_Cancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	err := sleepWithContext(ctx, time.Second)
	elapsed := time.Since(start)

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.True(t, elapsed < 100*time.Millisecond, "should return quickly when context is cancelled")
}

func TestUnaryClientInterceptor_ImmediateSuccess(t *testing.T) {
	interceptor := UnaryClientInterceptor()
	mock := &mockInvoker{errs: []error{nil}}

	err := interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke)

	assert.NoError(t, err)
	assert.Equal(t, 1, mock.callCount)
}

func TestUnaryClientInterceptor_SuccessAfterRetries(t *testing.T) {
	interceptor := UnaryClientInterceptor(
		MaxRetries(3),
		BackoffFunc(func(ctx context.Context, attempt uint) time.Duration {
			return 1 * time.Millisecond
		}),
	)
	mock := &mockInvoker{errs: []error{
		status.Error(codes.Unavailable, "unavailable"),
		status.Error(codes.Unavailable, "unavailable"),
		nil,
	}}

	err := interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke)

	assert.NoError(t, err)
	assert.Equal(t, 3, mock.callCount)
}

func TestUnaryClientInterceptor_NonRetryableError(t *testing.T) {
	interceptor := UnaryClientInterceptor()
	mock := &mockInvoker{errs: []error{
		status.Error(codes.NotFound, "not found"),
	}}

	err := interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke)

	assert.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
	assert.Equal(t, 1, mock.callCount)
}

func TestUnaryClientInterceptor_MaxRetriesExceeded(t *testing.T) {
	interceptor := UnaryClientInterceptor(
		MaxRetries(2),
		BackoffFunc(func(ctx context.Context, attempt uint) time.Duration {
			return 1 * time.Millisecond
		}),
	)
	mock := &mockInvoker{errs: []error{
		status.Error(codes.Unavailable, "unavailable 1"),
		status.Error(codes.Unavailable, "unavailable 2"),
		status.Error(codes.Unavailable, "unavailable 3"),
	}}

	err := interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke)

	assert.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
	assert.Equal(t, 3, mock.callCount)
}

func TestUnaryClientInterceptor_ContextCancelledDuringBackoff(t *testing.T) {
	interceptor := UnaryClientInterceptor(
		MaxRetries(3),
		BackoffFunc(func(ctx context.Context, attempt uint) time.Duration {
			return time.Second
		}),
	)
	mock := &mockInvoker{errs: []error{
		status.Error(codes.Unavailable, "unavailable"),
	}}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := interceptor(ctx, "/test/method", nil, nil, nil, mock.invoke)

	assert.Error(t, err)
	assert.Equal(t, codes.Canceled, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "context canceled during backoff")
	assert.Equal(t, 1, mock.callCount)
}

func TestUnaryClientInterceptor_ContextCancelledBeforeRetry(t *testing.T) {
	interceptor := UnaryClientInterceptor(
		MaxRetries(10),
		BackoffFunc(func(ctx context.Context, attempt uint) time.Duration {
			return 50 * time.Millisecond
		}),
	)

	callCount := 0
	invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		callCount++
		return status.Error(codes.Unavailable, "unavailable")
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(80 * time.Millisecond)
		cancel()
	}()

	err := interceptor(ctx, "/test/method", nil, nil, nil, invoker)

	assert.Error(t, err)
	assert.Equal(t, codes.Canceled, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "context canceled")
}

func TestUnaryClientInterceptor_PerCallTimeout_Success(t *testing.T) {
	interceptor := UnaryClientInterceptor(
		PerCallTimeout(100 * time.Millisecond),
	)
	mock := &mockInvoker{errs: []error{nil}}

	err := interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke)

	assert.NoError(t, err)
	assert.Equal(t, 1, mock.callCount)
}

func TestUnaryClientInterceptor_PerCallTimeout_Timeout(t *testing.T) {
	interceptor := UnaryClientInterceptor(
		MaxRetries(1),
		PerCallTimeout(50 * time.Millisecond),
		BackoffFunc(func(ctx context.Context, attempt uint) time.Duration {
			return 1 * time.Millisecond
		}),
	)

	callCount := 0
	invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		callCount++
		<-ctx.Done()
		return status.Error(codes.DeadlineExceeded, "timeout")
	}

	err := interceptor(context.Background(), "/test/method", nil, nil, nil, invoker)

	assert.Error(t, err)
	assert.Equal(t, codes.DeadlineExceeded, status.Code(err))
	assert.Equal(t, 2, callCount)
}

func TestUnaryClientInterceptor_ContextPropagation(t *testing.T) {
	interceptor := UnaryClientInterceptor()
	mock := &mockInvoker{errs: []error{nil}}

	ctx := context.WithValue(context.Background(), "key", "value")
	err := interceptor(ctx, "/test/method", nil, nil, nil, mock.invoke)

	assert.NoError(t, err)
	assert.Equal(t, ctx, mock.lastCtx)
	assert.Equal(t, "/test/method", mock.lastMethod)
}

func TestUnaryClientInterceptor_ZeroRetries(t *testing.T) {
	interceptor := UnaryClientInterceptor(MaxRetries(0))
	mock := &mockInvoker{errs: []error{
		status.Error(codes.Unavailable, "unavailable"),
	}}

	err := interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke)

	assert.Error(t, err)
	assert.Equal(t, 1, mock.callCount)
}

func TestStreamClientInterceptor_ImmediateSuccess(t *testing.T) {
	interceptor := StreamClientInterceptor()
	mockStream := &mockClientStream{}
	mock := &mockStreamer{clientStream: mockStream}

	stream, err := interceptor(context.Background(), nil, nil, "/test/method", mock.stream)

	assert.NoError(t, err)
	assert.NotNil(t, stream)
	assert.Equal(t, 1, mock.callCount)
}

func TestStreamClientInterceptor_SuccessAfterRetries(t *testing.T) {
	interceptor := StreamClientInterceptor(
		MaxRetries(3),
		BackoffFunc(func(ctx context.Context, attempt uint) time.Duration {
			return 1 * time.Millisecond
		}),
	)
	mockStream := &mockClientStream{}
	mock := &mockStreamer{
		errs: []error{
			status.Error(codes.Unavailable, "unavailable"),
			status.Error(codes.Unavailable, "unavailable"),
		},
		clientStream: mockStream,
	}

	stream, err := interceptor(context.Background(), nil, nil, "/test/method", mock.stream)

	assert.NoError(t, err)
	assert.NotNil(t, stream)
	assert.Equal(t, 3, mock.callCount)
}

func TestStreamClientInterceptor_NonRetryableError(t *testing.T) {
	interceptor := StreamClientInterceptor()
	mock := &mockStreamer{errs: []error{
		status.Error(codes.NotFound, "not found"),
	}}

	stream, err := interceptor(context.Background(), nil, nil, "/test/method", mock.stream)

	assert.Error(t, err)
	assert.Nil(t, stream)
	assert.Equal(t, codes.NotFound, status.Code(err))
	assert.Equal(t, 1, mock.callCount)
}

func TestStreamClientInterceptor_MaxRetriesExceeded(t *testing.T) {
	interceptor := StreamClientInterceptor(
		MaxRetries(2),
		BackoffFunc(func(ctx context.Context, attempt uint) time.Duration {
			return 1 * time.Millisecond
		}),
	)
	mock := &mockStreamer{errs: []error{
		status.Error(codes.Unavailable, "unavailable 1"),
		status.Error(codes.Unavailable, "unavailable 2"),
		status.Error(codes.Unavailable, "unavailable 3"),
	}}

	stream, err := interceptor(context.Background(), nil, nil, "/test/method", mock.stream)

	assert.Error(t, err)
	assert.Nil(t, stream)
	assert.Equal(t, codes.Unavailable, status.Code(err))
	assert.Equal(t, 3, mock.callCount)
}

func TestStreamClientInterceptor_ContextCancelledDuringBackoff(t *testing.T) {
	interceptor := StreamClientInterceptor(
		MaxRetries(3),
		BackoffFunc(func(ctx context.Context, attempt uint) time.Duration {
			return time.Second
		}),
	)
	mock := &mockStreamer{errs: []error{
		status.Error(codes.Unavailable, "unavailable"),
	}}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	stream, err := interceptor(ctx, nil, nil, "/test/method", mock.stream)

	assert.Error(t, err)
	assert.Nil(t, stream)
	assert.Equal(t, codes.Canceled, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "context canceled during backoff")
	assert.Equal(t, 1, mock.callCount)
}

func TestStreamClientInterceptor_ContextCancelledBeforeRetry(t *testing.T) {
	interceptor := StreamClientInterceptor(
		MaxRetries(10),
		BackoffFunc(func(ctx context.Context, attempt uint) time.Duration {
			return 50 * time.Millisecond
		}),
	)

	callCount := 0
	streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		callCount++
		return nil, status.Error(codes.Unavailable, "unavailable")
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(80 * time.Millisecond)
		cancel()
	}()

	stream, err := interceptor(ctx, nil, nil, "/test/method", streamer)

	assert.Error(t, err)
	assert.Nil(t, stream)
	assert.Equal(t, codes.Canceled, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "context canceled")
}

func TestStreamClientInterceptor_PerCallTimeout_Success(t *testing.T) {
	interceptor := StreamClientInterceptor(
		PerCallTimeout(100 * time.Millisecond),
	)
	mockStream := &mockClientStream{}
	mock := &mockStreamer{clientStream: mockStream}

	stream, err := interceptor(context.Background(), nil, nil, "/test/method", mock.stream)

	assert.NoError(t, err)
	assert.NotNil(t, stream)
	assert.Equal(t, 1, mock.callCount)
}

func TestStreamClientInterceptor_PerCallTimeout_Timeout(t *testing.T) {
	interceptor := StreamClientInterceptor(
		MaxRetries(1),
		PerCallTimeout(50 * time.Millisecond),
		BackoffFunc(func(ctx context.Context, attempt uint) time.Duration {
			return 1 * time.Millisecond
		}),
	)

	callCount := 0
	streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		callCount++
		<-ctx.Done()
		return nil, status.Error(codes.DeadlineExceeded, "timeout")
	}

	stream, err := interceptor(context.Background(), nil, nil, "/test/method", streamer)

	assert.Error(t, err)
	assert.Nil(t, stream)
	assert.Equal(t, codes.DeadlineExceeded, status.Code(err))
	assert.Equal(t, 2, callCount)
}

func TestStreamClientInterceptor_ContextPropagation(t *testing.T) {
	interceptor := StreamClientInterceptor()
	mockStream := &mockClientStream{}
	mock := &mockStreamer{clientStream: mockStream}

	ctx := context.WithValue(context.Background(), "key", "value")
	stream, err := interceptor(ctx, nil, nil, "/test/method", mock.stream)

	assert.NoError(t, err)
	assert.NotNil(t, stream)
	assert.Equal(t, ctx, mock.lastCtx)
	assert.Equal(t, "/test/method", mock.lastMethod)
}

func TestStreamClientInterceptor_ZeroRetries(t *testing.T) {
	interceptor := StreamClientInterceptor(MaxRetries(0))
	mock := &mockStreamer{errs: []error{
		status.Error(codes.Unavailable, "unavailable"),
	}}

	stream, err := interceptor(context.Background(), nil, nil, "/test/method", mock.stream)

	assert.Error(t, err)
	assert.Nil(t, stream)
	assert.Equal(t, 1, mock.callCount)
}

func TestClientStreamWithCancel_RecvMsg_Success(t *testing.T) {
	mockStream := &mockClientStream{recvErr: nil}
	cancelCalled := false
	cancel := func() {
		cancelCalled = true
	}

	stream := &clientStreamWithCancel{ClientStream: mockStream, cancel: cancel}
	err := stream.RecvMsg(nil)

	assert.NoError(t, err)
	assert.False(t, cancelCalled, "cancel should not be called on success")
}

func TestClientStreamWithCancel_RecvMsg_EOF(t *testing.T) {
	mockStream := &mockClientStream{recvErr: io.EOF}
	cancelCalled := false
	cancel := func() {
		cancelCalled = true
	}

	stream := &clientStreamWithCancel{ClientStream: mockStream, cancel: cancel}
	err := stream.RecvMsg(nil)

	assert.Equal(t, io.EOF, err)
	assert.True(t, cancelCalled, "cancel should be called on EOF")
}

func TestClientStreamWithCancel_RecvMsg_Error(t *testing.T) {
	mockStream := &mockClientStream{recvErr: status.Error(codes.Internal, "internal error")}
	cancelCalled := false
	cancel := func() {
		cancelCalled = true
	}

	stream := &clientStreamWithCancel{ClientStream: mockStream, cancel: cancel}
	err := stream.RecvMsg(nil)

	assert.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.True(t, cancelCalled, "cancel should be called on error")
}

func TestClientStreamWithCancel_CloseSend(t *testing.T) {
	mockStream := &mockClientStream{}
	cancelCalled := false
	cancel := func() {
		cancelCalled = true
	}

	stream := &clientStreamWithCancel{ClientStream: mockStream, cancel: cancel}
	err := stream.CloseSend()

	assert.NoError(t, err)
	assert.True(t, cancelCalled, "cancel should be called on CloseSend")
}

func TestClientStreamWithCancel_RecvMsg_MultipleCalls(t *testing.T) {
	mockStream := &mockClientStream{recvErr: status.Error(codes.Internal, "error")}
	cancelCount := 0
	cancel := func() {
		cancelCount++
	}

	stream := &clientStreamWithCancel{ClientStream: mockStream, cancel: cancel}

	_ = stream.RecvMsg(nil)
	_ = stream.RecvMsg(nil)
	_ = stream.RecvMsg(nil)

	assert.Equal(t, 3, cancelCount, "cancel should be called on each error")
	assert.Equal(t, 3, mockStream.recvCalled)
}
