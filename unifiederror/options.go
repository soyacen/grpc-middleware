// Package unifiederror 提供gRPC错误处理的配置选项
package unifiederror

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// options 存储错误处理的配置选项
type options struct {
	// errorFunc 错误处理函数，用于将普通错误转换为gRPC状态
	errorFunc func(err error) *status.Status
}

// apply 应用所有配置选项
func (o *options) apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

// Option 表示错误处理的配置选项函数
type Option func(o *options)

// defaultOptions 创建默认的错误处理选项
// 默认错误处理逻辑：
// - nil错误返回nil
// - context.DeadlineExceeded转换为DeadlineExceeded状态
// - context.Canceled转换为Canceled状态
// - 实现GRPCStatus接口的错误直接返回其状态
// - 其他错误转换为Unknown状态
func defaultOptions() *options {
	return &options{
		errorFunc: func(err error) *status.Status {
			switch err {
			case nil:
				return nil
			case context.DeadlineExceeded:
				return status.New(codes.DeadlineExceeded, err.Error())
			case context.Canceled:
				return status.New(codes.Canceled, err.Error())
			default:
				if se, ok := err.(interface{ GRPCStatus() *status.Status }); ok {
					return se.GRPCStatus()
				} else {
					return status.New(codes.Unknown, err.Error())
				}
			}
		},
	}
}

// ErrorFunc 设置自定义的错误处理函数
//
// 参数:
//   - errorFunc: 自定义的错误处理函数
//
// 返回值:
//   - Option: 配置选项函数
func ErrorFunc(errorFunc func(err error) *status.Status) Option {
	return func(o *options) {
		o.errorFunc = errorFunc
	}
}
