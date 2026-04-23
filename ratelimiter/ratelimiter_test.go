package ratelimiter

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestErrLimitExceeded(t *testing.T) {
	tests := []struct {
		name     string
		wantCode codes.Code
		wantMsg  string
	}{
		{
			name:     "verify_error_code_is_ResourceExhausted",
			wantCode: codes.ResourceExhausted,
			wantMsg:  "ratelimiter: rate limit exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ErrLimitExceeded == nil {
				t.Fatal("ErrLimitExceeded should not be nil")
			}
			st, ok := status.FromError(ErrLimitExceeded)
			if !ok {
				t.Fatal("ErrLimitExceeded should be a gRPC status error")
			}
			if st.Code() != tt.wantCode {
				t.Errorf("ErrLimitExceeded code = %v, want %v", st.Code(), tt.wantCode)
			}
			if st.Message() != tt.wantMsg {
				t.Errorf("ErrLimitExceeded message = %q, want %q", st.Message(), tt.wantMsg)
			}
		})
	}
}

func TestRateLimiter_Interface(t *testing.T) {
	tests := []struct {
		name         string
		description  string
	}{
		{
			name:         "interface_exists",
			description:  "RateLimiter接口应该存在并定义Allow方法",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var _ RateLimiter = (*mockRateLimiter)(nil)
		})
	}
}

type mockRateLimiter struct {
	allowCalled bool
}

func (m *mockRateLimiter) Allow() (done func(DoneInfo), err error) {
	m.allowCalled = true
	return func(DoneInfo) {}, nil
}

func TestDoneInfo_Struct(t *testing.T) {
	tests := []struct {
		name     string
		doneInfo DoneInfo
		wantErr  error
	}{
		{
			name:     "empty_DoneInfo",
			doneInfo: DoneInfo{},
			wantErr:  nil,
		},
		{
			name:     "DoneInfo_with_error",
			doneInfo: DoneInfo{Err: status.Error(codes.Internal, "test error")},
			wantErr:  status.Error(codes.Internal, "test error"),
		},
		{
			name:     "DoneInfo_with_nil_error",
			doneInfo: DoneInfo{Err: nil},
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doneInfo.Err != tt.wantErr {
				if tt.doneInfo.Err == nil || tt.wantErr == nil {
					t.Errorf("DoneInfo.Err = %v, want %v", tt.doneInfo.Err, tt.wantErr)
				} else if tt.doneInfo.Err.Error() != tt.wantErr.Error() {
					t.Errorf("DoneInfo.Err = %v, want %v", tt.doneInfo.Err, tt.wantErr)
				}
			}
		})
	}
}
