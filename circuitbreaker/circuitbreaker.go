package circuitbreaker

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrCircuitBreakerOpen 熔断器触发时返回的错误
var ErrCircuitBreakerOpen = status.Error(codes.ResourceExhausted, "circuitbreaker: adaptive throttling, request dropped")

// CircuitBreaker 熔断器接口，实现自适应节流算法
type CircuitBreaker interface {
	// Allow 判断请求是否被允许
	Allow() bool

	// MarkSuccess 标记请求成功
	MarkSuccess()

	// MarkFailure 标记请求失败
	MarkFailure()
}
