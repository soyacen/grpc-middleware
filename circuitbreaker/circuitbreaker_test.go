package circuitbreaker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestErrCircuitBreakerOpen(t *testing.T) {
	tests := []struct {
		name    string
		wantMsg string
	}{
		{
			name:    "error_contains_expected_message",
			wantMsg: "circuitbreaker: adaptive throttling, request dropped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ErrCircuitBreakerOpen
			assert.Error(t, got)
			assert.Contains(t, got.Error(), tt.wantMsg)
		})
	}
}

func TestCircuitBreaker_Interface(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "interface_has_allow_method"},
		{name: "interface_has_mark_success_method"},
		{name: "interface_has_mark_failure_method"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var _ CircuitBreaker = (*testCircuitBreaker)(nil)
		})
	}
}

type testCircuitBreaker struct{}

func (m *testCircuitBreaker) Allow() bool  { return true }
func (m *testCircuitBreaker) MarkSuccess() {}
func (m *testCircuitBreaker) MarkFailure() {}

func TestErrCircuitBreakerOpen_GRPCStatus(t *testing.T) {
	tests := []struct {
		name        string
		wantCode    codes.Code
		wantMessage string
	}{
		{
			name:        "grpc_status_code_is_resource_exhausted",
			wantCode:    codes.ResourceExhausted,
			wantMessage: "circuitbreaker: adaptive throttling, request dropped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, ok := status.FromError(ErrCircuitBreakerOpen)
			assert.True(t, ok)
			assert.Equal(t, tt.wantCode, st.Code())
			assert.Contains(t, st.Message(), tt.wantMessage)
		})
	}
}
