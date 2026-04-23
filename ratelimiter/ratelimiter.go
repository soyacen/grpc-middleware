package ratelimiter

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrLimitExceeded 限流触发时返回的错误
var ErrLimitExceeded = status.Error(codes.ResourceExhausted, "ratelimiter: rate limit exceeded")

// DoneInfo 请求完成时传递的信息
type DoneInfo struct {
	// Err 请求执行返回的错误
	Err error
}

// RateLimiter 限流器接口
type RateLimiter interface {
	// Allow 检查是否允许执行请求
	// 返回完成回调函数和错误（如果请求被拒绝）
	Allow() (done func(DoneInfo), err error)
}
