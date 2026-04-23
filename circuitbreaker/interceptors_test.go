package circuitbreaker

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockInvoker struct {
	err        error
	callCount  int
	lastCtx    context.Context
	lastMethod string
}

func (m *mockInvoker) invoke(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
	m.callCount++
	m.lastCtx = ctx
	m.lastMethod = method
	return m.err
}

func TestUnary_AllowAndSuccess(t *testing.T) {
	interceptor := UnaryClientInterceptor(WithK(2.0))
	mock := &mockInvoker{err: nil}

	err := interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke)

	assert.NoError(t, err)
	assert.Equal(t, 1, mock.callCount)
}

func TestUnary_AllowAndFailure(t *testing.T) {
	interceptor := UnaryClientInterceptor(WithK(2.0))
	mock := &mockInvoker{err: status.Error(codes.Internal, "internal error")}

	err := interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke)

	assert.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Equal(t, 1, mock.callCount)
}

func TestUnary_Dropped(t *testing.T) {
	interceptor := UnaryClientInterceptor(WithK(1.0), WithWindow(time.Second), WithBuckets(10))
	failMock := &mockInvoker{err: status.Error(codes.Internal, "error")}

	for i := 0; i < 100; i++ {
		_ = interceptor(context.Background(), "/test/method", nil, nil, nil, failMock.invoke)
	}

	successMock := &mockInvoker{err: nil}
	dropped := 0
	for i := 0; i < 100; i++ {
		err := interceptor(context.Background(), "/test/method", nil, nil, nil, successMock.invoke)
		if status.Code(err) == codes.ResourceExhausted {
			dropped++
		}
	}

	assert.Greater(t, dropped, 30, "should drop requests when circuit is open")
}

func TestUnary_DropDoesNotCallInvoker(t *testing.T) {
	interceptor := UnaryClientInterceptor(WithK(1.0), WithWindow(time.Second), WithBuckets(10))
	failMock := &mockInvoker{err: status.Error(codes.Internal, "error")}

	for i := 0; i < 100; i++ {
		_ = interceptor(context.Background(), "/test/method", nil, nil, nil, failMock.invoke)
	}

	successMock := &mockInvoker{err: nil}
	droppedCount := 0
	allowedCount := 0
	for i := 0; i < 50; i++ {
		err := interceptor(context.Background(), "/test/method", nil, nil, nil, successMock.invoke)
		if status.Code(err) == codes.ResourceExhausted {
			droppedCount++
		} else {
			allowedCount++
		}
	}

	assert.Greater(t, droppedCount, 0, "should drop some requests")
	assert.Equal(t, allowedCount, successMock.callCount, "only allowed requests should call invoker")
}

func TestUnary_DeadlineExceeded_MarkFailure(t *testing.T) {
	interceptor := UnaryClientInterceptor(WithK(2.0))
	mock := &mockInvoker{err: status.Error(codes.DeadlineExceeded, "timeout")}

	err := interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke)
	assert.Equal(t, codes.DeadlineExceeded, status.Code(err))
	assert.Equal(t, 1, mock.callCount)
}

func TestUnary_Internal_MarkFailure(t *testing.T) {
	interceptor := UnaryClientInterceptor(WithK(2.0))
	mock := &mockInvoker{err: status.Error(codes.Internal, "internal error")}

	err := interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Equal(t, 1, mock.callCount)
}

func TestUnary_Unavailable_MarkFailure(t *testing.T) {
	interceptor := UnaryClientInterceptor(WithK(2.0))
	mock := &mockInvoker{err: status.Error(codes.Unavailable, "unavailable")}

	err := interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke)
	assert.Equal(t, codes.Unavailable, status.Code(err))
	assert.Equal(t, 1, mock.callCount)
}

func TestUnary_ResourceExhausted_MarkFailure(t *testing.T) {
	interceptor := UnaryClientInterceptor(WithK(2.0))
	mock := &mockInvoker{err: status.Error(codes.ResourceExhausted, "exhausted")}

	err := interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke)
	assert.Equal(t, codes.ResourceExhausted, status.Code(err))
	assert.Equal(t, 1, mock.callCount)
}

func TestUnary_OK_MarkSuccess(t *testing.T) {
	interceptor := UnaryClientInterceptor(WithK(2.0))
	mock := &mockInvoker{err: nil}

	err := interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke)
	assert.NoError(t, err)
	assert.Equal(t, 1, mock.callCount)
}

func TestUnary_NotFound_MarkSuccess(t *testing.T) {
	interceptor := UnaryClientInterceptor(WithK(2.0))
	mock := &mockInvoker{err: status.Error(codes.NotFound, "not found")}

	err := interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke)
	assert.Equal(t, codes.NotFound, status.Code(err))
	assert.Equal(t, 1, mock.callCount)
}

func TestUnary_InvalidArgument_MarkSuccess(t *testing.T) {
	interceptor := UnaryClientInterceptor(WithK(2.0))
	mock := &mockInvoker{err: status.Error(codes.InvalidArgument, "invalid")}

	err := interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Equal(t, 1, mock.callCount)
}

func TestUnary_NonStatusError_MarkFailure(t *testing.T) {
	interceptor := UnaryClientInterceptor(WithK(2.0))
	mock := &mockInvoker{err: errors.New("non-status error")}

	err := interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke)
	assert.Equal(t, "non-status error", err.Error())
	assert.Equal(t, 1, mock.callCount)
}

func TestUnary_FailureCodesTriggerDrop(t *testing.T) {
	interceptor := UnaryClientInterceptor(WithK(1.0), WithWindow(time.Second), WithBuckets(10))
	mock := &mockInvoker{err: status.Error(codes.Internal, "error")}

	for i := 0; i < 100; i++ {
		_ = interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke)
	}

	dropped := 0
	for i := 0; i < 100; i++ {
		err := interceptor(context.Background(), "/test/method", nil, nil, nil, mock.invoke)
		if status.Code(err) == codes.ResourceExhausted {
			dropped++
		}
	}
	assert.Greater(t, dropped, 50, "repeated failures should eventually trigger drops")
}

func TestUnary_ContextPropagation(t *testing.T) {
	interceptor := UnaryClientInterceptor(WithK(2.0))
	mock := &mockInvoker{err: nil}

	ctx := context.WithValue(context.Background(), "key", "value")
	err := interceptor(ctx, "/test/method", nil, nil, nil, mock.invoke)

	assert.NoError(t, err)
	assert.Equal(t, ctx, mock.lastCtx, "context should be propagated")
	assert.Equal(t, "/test/method", mock.lastMethod, "method should be propagated")
}

func TestUnary_MultipleMethods(t *testing.T) {
	interceptor := UnaryClientInterceptor(WithK(2.0))

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

type mockClientStream struct {
	grpc.ClientStream
	recvErr    error
	recvCalled int
}

func (m *mockClientStream) RecvMsg(msg interface{}) error {
	m.recvCalled++
	return m.recvErr
}

func TestStreamClientInterceptor_AllowAndSuccess(t *testing.T) {
	interceptor := StreamClientInterceptor(WithK(2.0))

	streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		return &mockClientStream{}, nil
	}

	stream, err := interceptor(context.Background(), nil, nil, "/test/method", streamer)

	assert.NoError(t, err)
	assert.NotNil(t, stream)
}

func TestStreamClientInterceptor_Dropped(t *testing.T) {
	interceptor := StreamClientInterceptor(WithK(1.0), WithWindow(time.Second), WithBuckets(10))

	failStreamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		return nil, status.Error(codes.Internal, "error")
	}

	for i := 0; i < 100; i++ {
		_, _ = interceptor(context.Background(), nil, nil, "/test/method", failStreamer)
	}

	successStreamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		return &mockClientStream{}, nil
	}

	dropped := 0
	for i := 0; i < 100; i++ {
		_, err := interceptor(context.Background(), nil, nil, "/test/method", successStreamer)
		if status.Code(err) == codes.ResourceExhausted {
			dropped++
		}
	}

	assert.Greater(t, dropped, 30, "should drop requests when circuit is open")
}

func TestStreamClientInterceptor_StreamerError_MarkFailure(t *testing.T) {
	interceptor := StreamClientInterceptor(WithK(2.0))

	codes := []codes.Code{
		codes.DeadlineExceeded,
		codes.Internal,
		codes.Unavailable,
		codes.ResourceExhausted,
	}

	for _, code := range codes {
		streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
			return nil, status.Error(code, "error")
		}

		_, err := interceptor(context.Background(), nil, nil, "/test/method", streamer)
		assert.Error(t, err)
	}
}

func TestStreamClientInterceptor_StreamerError_MarkSuccess(t *testing.T) {
	interceptor := StreamClientInterceptor(WithK(2.0))

	codes := []codes.Code{
		codes.NotFound,
		codes.InvalidArgument,
		codes.PermissionDenied,
	}

	for _, code := range codes {
		streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
			return nil, status.Error(code, "error")
		}

		_, err := interceptor(context.Background(), nil, nil, "/test/method", streamer)
		assert.Error(t, err)
	}
}

func TestStreamClientInterceptor_StreamerError_NonStatus(t *testing.T) {
	interceptor := StreamClientInterceptor(WithK(2.0))

	streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		return nil, errors.New("custom error")
	}

	_, err := interceptor(context.Background(), nil, nil, "/test/method", streamer)
	assert.Error(t, err)
	assert.Equal(t, "custom error", err.Error())
}

func TestWrappedClientStream_RecvMsg_Success(t *testing.T) {
	breaker := defaultOptions().apply(WithK(2.0)).init().newCircuitBreaker()
	mockStream := &mockClientStream{recvErr: nil}

	wrapped := &wrappedClientStream{
		ClientStream: mockStream,
		breaker:      breaker,
	}

	err := wrapped.RecvMsg(nil)
	assert.NoError(t, err)
}

func TestWrappedClientStream_RecvMsg_EOF(t *testing.T) {
	breaker := defaultOptions().apply(WithK(2.0)).init().newCircuitBreaker()
	mockStream := &mockClientStream{recvErr: io.EOF}

	wrapped := &wrappedClientStream{
		ClientStream: mockStream,
		breaker:      breaker,
	}

	err := wrapped.RecvMsg(nil)
	assert.Equal(t, io.EOF, err)
}

func TestWrappedClientStream_RecvMsg_InternalError(t *testing.T) {
	breaker := defaultOptions().apply(WithK(2.0)).init().newCircuitBreaker()
	mockStream := &mockClientStream{recvErr: status.Error(codes.Internal, "internal error")}

	wrapped := &wrappedClientStream{
		ClientStream: mockStream,
		breaker:      breaker,
	}

	err := wrapped.RecvMsg(nil)
	assert.Error(t, err)
}

func TestWrappedClientStream_RecvMsg_NotFound(t *testing.T) {
	breaker := defaultOptions().apply(WithK(2.0)).init().newCircuitBreaker()
	mockStream := &mockClientStream{recvErr: status.Error(codes.NotFound, "not found")}

	wrapped := &wrappedClientStream{
		ClientStream: mockStream,
		breaker:      breaker,
	}

	err := wrapped.RecvMsg(nil)
	assert.Error(t, err)
}

func TestWrappedClientStream_RecvMsg_NonStatusError(t *testing.T) {
	breaker := defaultOptions().apply(WithK(2.0)).init().newCircuitBreaker()
	mockStream := &mockClientStream{recvErr: errors.New("custom error")}

	wrapped := &wrappedClientStream{
		ClientStream: mockStream,
		breaker:      breaker,
	}

	err := wrapped.RecvMsg(nil)
	assert.Error(t, err)
	assert.Equal(t, "custom error", err.Error())
}

func TestWrappedClientStream_RecvMsg_OnlyMarksOnce(t *testing.T) {
	breaker := defaultOptions().apply(WithK(2.0)).init().newCircuitBreaker()
	mockStream := &mockClientStream{recvErr: status.Error(codes.Internal, "error")}

	wrapped := &wrappedClientStream{
		ClientStream: mockStream,
		breaker:      breaker,
	}

	_ = wrapped.RecvMsg(nil)
	_ = wrapped.RecvMsg(nil)
	_ = wrapped.RecvMsg(nil)

	assert.Equal(t, 3, mockStream.recvCalled)
}
