package unifiederror

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type mockUnaryHandler struct {
	resp interface{}
	err  error
}

func (m *mockUnaryHandler) handle(ctx context.Context, req interface{}) (interface{}, error) {
	return m.resp, m.err
}

type mockServerStream struct{}

func (m *mockServerStream) SetHeader(md metadata.MD) error  { return nil }
func (m *mockServerStream) SendHeader(md metadata.MD) error { return nil }
func (m *mockServerStream) SetTrailer(md metadata.MD)       {}
func (m *mockServerStream) Context() context.Context        { return context.Background() }
func (m *mockServerStream) SendMsg(v interface{}) error     { return nil }
func (m *mockServerStream) RecvMsg(v interface{}) error     { return nil }

type mockStreamHandler struct {
	err error
}

func (m *mockStreamHandler) handle(srv interface{}, stream grpc.ServerStream) error {
	return m.err
}

type grpcStatusError struct {
	code    codes.Code
	message string
}

func (e *grpcStatusError) Error() string {
	return e.message
}

func (e *grpcStatusError) GRPCStatus() *status.Status {
	return status.New(e.code, e.message)
}

func TestUnaryServerInterceptor(t *testing.T) {
	tests := []struct {
		name         string
		handlerResp  interface{}
		handlerErr   error
		wantResp     interface{}
		wantErrCode  codes.Code
		wantErrMsg   string
		wantNilErr   bool
	}{
		{
			name:        "nil_error_passes_through",
			handlerResp: "success_response",
			handlerErr:  nil,
			wantResp:    "success_response",
			wantNilErr:  true,
		},
		{
			name:        "nil_error_with_nil_response",
			handlerResp: nil,
			handlerErr:  nil,
			wantResp:    nil,
			wantNilErr:  true,
		},
		{
			name:        "deadline_exceeded_converted",
			handlerResp: nil,
			handlerErr:  context.DeadlineExceeded,
			wantResp:    nil,
			wantErrCode: codes.DeadlineExceeded,
			wantErrMsg:  context.DeadlineExceeded.Error(),
		},
		{
			name:        "canceled_converted",
			handlerResp: nil,
			handlerErr:  context.Canceled,
			wantResp:    nil,
			wantErrCode: codes.Canceled,
			wantErrMsg:  context.Canceled.Error(),
		},
		{
			name:        "grpc_status_error_preserved",
			handlerResp: nil,
			handlerErr:  &grpcStatusError{code: codes.InvalidArgument, message: "invalid argument"},
			wantResp:    nil,
			wantErrCode: codes.InvalidArgument,
			wantErrMsg:  "invalid argument",
		},
		{
			name:        "grpc_status_error_internal",
			handlerResp: nil,
			handlerErr:  &grpcStatusError{code: codes.Internal, message: "internal error"},
			wantResp:    nil,
			wantErrCode: codes.Internal,
			wantErrMsg:  "internal error",
		},
		{
			name:        "plain_error_to_unknown",
			handlerResp: nil,
			handlerErr:  errors.New("plain error"),
			wantResp:    nil,
			wantErrCode: codes.Unknown,
			wantErrMsg:  "plain error",
		},
		{
			name:        "wrapped_error_to_unknown",
			handlerResp: nil,
			handlerErr:  errors.New("wrapped error"),
			wantResp:    nil,
			wantErrCode: codes.Unknown,
			wantErrMsg:  "wrapped error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &mockUnaryHandler{resp: tt.handlerResp, err: tt.handlerErr}
			interceptor := UnaryServerInterceptor()
			resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler.handle)

			assert.Equal(t, tt.wantResp, resp)

			if tt.wantNilErr {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok, "error should be a gRPC status error")
				assert.Equal(t, tt.wantErrCode, st.Code())
				assert.Equal(t, tt.wantErrMsg, st.Message())
			}
		})
	}
}

func TestUnaryServerInterceptor_WithCustomErrorFunc(t *testing.T) {
	tests := []struct {
		name        string
		handlerErr  error
		wantResp    interface{}
		wantNilErr  bool
		wantErrCode codes.Code
		wantErrMsg  string
	}{
		{
			name:       "custom_error_func_nil_returns_nil",
			handlerErr: nil,
			wantResp:   "resp",
			wantNilErr: true,
		},
		{
			name:        "custom_error_func_converts_error",
			handlerErr:  errors.New("some error"),
			wantErrCode: codes.Unauthenticated,
			wantErrMsg:  "custom: some error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &mockUnaryHandler{resp: "resp", err: tt.handlerErr}
			customErrorFunc := func(err error) *status.Status {
				if err == nil {
					return nil
				}
				return status.New(codes.Unauthenticated, "custom: "+err.Error())
			}
			interceptor := UnaryServerInterceptor(ErrorFunc(customErrorFunc))
			resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler.handle)

			assert.Equal(t, tt.wantResp, resp)

			if tt.wantNilErr {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantErrCode, st.Code())
				assert.Equal(t, tt.wantErrMsg, st.Message())
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

			handler := grpc.UnaryHandler(func(c context.Context, req interface{}) (interface{}, error) {
				receivedCtx = c
				return "resp", nil
			})
			interceptor := UnaryServerInterceptor()
			resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)

			assert.NoError(t, err)
			assert.Equal(t, "resp", resp)
			assert.Equal(t, "value", receivedCtx.Value("key"))
		})
	}
}

func TestUnaryServerInterceptor_ResponsePreserved(t *testing.T) {
	tests := []struct {
		name        string
		resp        interface{}
		wantResp    interface{}
	}{
		{"string_response", "hello", "hello"},
		{"struct_response", map[string]string{"key": "value"}, map[string]string{"key": "value"}},
		{"nil_response", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &mockUnaryHandler{resp: tt.resp, err: nil}
			interceptor := UnaryServerInterceptor()
			resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler.handle)

			assert.NoError(t, err)
			assert.Equal(t, tt.wantResp, resp)
		})
	}
}

func TestStreamServerInterceptor(t *testing.T) {
	tests := []struct {
		name        string
		handlerErr  error
		wantErrCode codes.Code
		wantErrMsg  string
		wantNilErr  bool
	}{
		{
			name:       "nil_error_passes_through",
			handlerErr: nil,
			wantNilErr: true,
		},
		{
			name:        "deadline_exceeded_converted",
			handlerErr:  context.DeadlineExceeded,
			wantErrCode: codes.DeadlineExceeded,
			wantErrMsg:  context.DeadlineExceeded.Error(),
		},
		{
			name:        "canceled_converted",
			handlerErr:  context.Canceled,
			wantErrCode: codes.Canceled,
			wantErrMsg:  context.Canceled.Error(),
		},
		{
			name:        "grpc_status_error_preserved",
			handlerErr:  &grpcStatusError{code: codes.NotFound, message: "not found"},
			wantErrCode: codes.NotFound,
			wantErrMsg:  "not found",
		},
		{
			name:        "grpc_status_error_permission_denied",
			handlerErr:  &grpcStatusError{code: codes.PermissionDenied, message: "permission denied"},
			wantErrCode: codes.PermissionDenied,
			wantErrMsg:  "permission denied",
		},
		{
			name:        "plain_error_to_unknown",
			handlerErr:  errors.New("stream error"),
			wantErrCode: codes.Unknown,
			wantErrMsg:  "stream error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &mockStreamHandler{err: tt.handlerErr}
			interceptor := StreamServerInterceptor()
			stream := &mockServerStream{}
			err := interceptor(nil, stream, &grpc.StreamServerInfo{}, handler.handle)

			if tt.wantNilErr {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok, "error should be a gRPC status error")
				assert.Equal(t, tt.wantErrCode, st.Code())
				assert.Equal(t, tt.wantErrMsg, st.Message())
			}
		})
	}
}

func TestStreamServerInterceptor_WithCustomErrorFunc(t *testing.T) {
	tests := []struct {
		name        string
		handlerErr  error
		wantNilErr  bool
		wantErrCode codes.Code
		wantErrMsg  string
	}{
		{
			name:       "custom_error_func_nil_returns_nil",
			handlerErr: nil,
			wantNilErr: true,
		},
		{
			name:        "custom_error_func_converts_error",
			handlerErr:  errors.New("stream error"),
			wantErrCode: codes.ResourceExhausted,
			wantErrMsg:  "custom stream: stream error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &mockStreamHandler{err: tt.handlerErr}
			customErrorFunc := func(err error) *status.Status {
				if err == nil {
					return nil
				}
				return status.New(codes.ResourceExhausted, "custom stream: "+err.Error())
			}
			interceptor := StreamServerInterceptor(ErrorFunc(customErrorFunc))
			stream := &mockServerStream{}
			err := interceptor(nil, stream, &grpc.StreamServerInfo{}, handler.handle)

			if tt.wantNilErr {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantErrCode, st.Code())
				assert.Equal(t, tt.wantErrMsg, st.Message())
			}
		})
	}
}

func TestStreamServerInterceptor_HandlerCalled(t *testing.T) {
	tests := []struct {
		name       string
		handlerErr error
	}{
		{"handler_called_with_nil_error", nil},
		{"handler_called_with_error", errors.New("error")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			handler := grpc.StreamHandler(func(srv interface{}, stream grpc.ServerStream) error {
				callCount++
				return tt.handlerErr
			})
			interceptor := StreamServerInterceptor()
			stream := &mockServerStream{}
			_ = interceptor(nil, stream, &grpc.StreamServerInfo{}, handler)

			assert.Equal(t, 1, callCount, "handler should be called exactly once")
		})
	}
}
