package circuitbreaker

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrCircuitBreakerOpen 当熔断器触发时返回的错误
var ErrCircuitBreakerOpen = status.Error(codes.ResourceExhausted, "circuitbreaker: adaptive throttling, request dropped")

// SREBreaker 实现Google SRE手册中的自适应节流算法
// 参考: https://sre.google/sre-book/handling-overload/#eq2101
type SREBreaker interface {
	// Allow 判断是否允许执行请求
	// 返回true表示允许执行，false表示应该拒绝请求
	Allow() bool

	// MarkSuccess 标记一次成功的请求
	// 在请求成功完成后调用此方法
	MarkSuccess()

	// MarkFailure 标记一次失败的请求
	// 在请求失败后调用此方法
	MarkFailure()
}
