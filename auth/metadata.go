// Package auth 提供gRPC认证相关的元数据处理功能
package auth

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	// headerAuthorize 定义认证头字段名
	headerAuthorize = "authorization"
)

// AuthFromMD 从gRPC元数据中提取认证令牌
// 支持标准的Authorization头格式: "Scheme Token"
//
// 参数:
//   - ctx: 请求上下文
//   - expectedScheme: 期望的认证方案(如"Bearer")
//
// 返回值:
//   - string: 提取到的令牌
//   - error: 认证错误
func AuthFromMD(ctx context.Context, expectedScheme string) (string, error) {
	// 从上下文中获取authorization头的值
	vals := metadata.ValueFromIncomingContext(ctx, headerAuthorize)
	if len(vals) == 0 {
		return "", status.Error(codes.Unauthenticated, "Request unauthenticated with "+expectedScheme)
	}
	// 分割scheme和token
	scheme, token, found := strings.Cut(vals[0], " ")
	if !found {
		return "", status.Error(codes.Unauthenticated, "Bad authorization string")
	}
	// 验证scheme是否匹配期望值(忽略大小写)
	if !strings.EqualFold(scheme, expectedScheme) {
		return "", status.Error(codes.Unauthenticated, "Request unauthenticated with "+expectedScheme)
	}
	return token, nil
}
