// Package log 提供gRPC日志记录的配置选项
package log

import (
	"context"
	"log/slog"
)

// LoggerFactory 定义日志记录器工厂函数类型
// 用于根据上下文创建特定的日志记录器实例
//
// 参数:
//   - ctx: 可能包含创建日志记录器所需信息的上下文
//
// 返回值:
//   - *slog.Logger: 日志记录器实例
//   - error: 如果日志记录器创建失败则返回错误
type LoggerFactory func(ctx context.Context) (*slog.Logger, error)

// options 存储访问日志中间件的配置选项
type options struct {
	// loggerFactory 用于创建日志记录器的工厂函数
	loggerFactory LoggerFactory
	// level 访问日志条目的日志级别
	level slog.Level
	// skip 用于确定是否跳过日志记录的函数
	skip func(fullMethodName string, err error) bool
}

// apply 将给定的选项应用到选项结构体中
//
// 参数:
//   - opts: 可变数量的选项函数
//
// 返回值:
//   - *options: 指向更新后的选项结构体的指针
func (o *options) apply(opts ...Option) *options {
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Option 定义用于配置访问日志中间件选项的函数类型
type Option func(o *options)

// defaultOptions 返回默认的配置选项
//
// 返回值:
//   - *options: 包含默认选项的结构体指针
func defaultOptions() *options {
	return &options{
		loggerFactory: func(_ context.Context) (*slog.Logger, error) {
			return slog.Default(), nil
		},
		level: slog.LevelInfo,
		skip: func(fullMethodName string, err error) bool {
			return false
		},
	}
}

// WithLoggerFactory 设置日志记录器工厂函数
//
// 参数:
//   - loggerFactory: 从上下文创建日志记录器的函数
//
// 返回值:
//   - Option: 设置日志记录器工厂选项的函数
func WithLoggerFactory(loggerFactory LoggerFactory) Option {
	return func(o *options) {
		o.loggerFactory = loggerFactory
	}
}

// WithLevel 设置访问日志条目的日志级别
//
// 参数:
//   - level: 访问日志的日志级别
//
// 返回值:
//   - Option: 设置日志级别选项的函数
func WithLevel(level slog.Level) Option {
	return func(o *options) {
		o.level = level
	}
}

// WithSkip 设置跳过函数以确定是否跳过日志记录
//
// 参数:
//   - skip: 确定是否跳过日志记录的函数
//
// 返回值:
//   - Option: 设置跳过选项的函数
func WithSkip(skip func(fullMethodName string, err error) bool) Option {
	return func(o *options) {
		o.skip = skip
	}
}
