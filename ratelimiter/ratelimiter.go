package ratelimiter

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrLimitExceeded 当限流触发时返回的错误
// 表示请求被限流器拒绝
var ErrLimitExceeded = status.Error(codes.ResourceExhausted, "ratelimiter: rate limit exceeded")

// DoneInfo 包含请求执行完成的信息
type DoneInfo struct {
	// Err 是请求处理器返回的错误
	Err error
}

// RateLimiter 定义限流器接口
type RateLimiter interface {
	// Allow 检查请求是否被允许
	// 返回完成回调函数和可能的错误
	// 如果请求不被允许，返回错误
	Allow() (done func(DoneInfo), err error)
}
