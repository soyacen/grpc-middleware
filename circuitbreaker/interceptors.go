// Package circuitbreaker 提供gRPC客户端熔断器中间件功能
// 实现基于Google SRE自适应节流算法的gRPC客户端拦截器
package circuitbreaker

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryClientInterceptor 创建一个使用 Google SRE 适应性节流算法的熔断器拦截器
// 该拦截器基于历史请求成功率动态调整请求通过率，防止服务过载
//
// 参数:
//   - opts: 可选的配置选项，用于自定义熔断器行为
//
// 返回:
//   - grpc.UnaryClientInterceptor: gRPC客户端一元调用拦截器函数
//
// 工作原理:
// 1. 在每次请求前调用 breaker.Allow() 判断是否允许执行
// 2. 请求成功时调用 breaker.MarkSuccess() 记录成功
// 3. 请求失败时调用 breaker.MarkFailure() 记录失败
// 4. 根据SRE算法动态计算拒绝概率，实现自适应节流
func UnaryClientInterceptor(opts ...Option) grpc.UnaryClientInterceptor {
	// 初始化熔断器配置并创建SRE熔断器实例
	breaker := defaultOptions().apply(opts...).init().newSREBreaker()

	return func(
		ctx context.Context, // 请求上下文
		method string, // gRPC方法名
		req interface{}, // 请求参数
		reply interface{}, // 响应结果
		cc *grpc.ClientConn, // gRPC客户端连接
		invoker grpc.UnaryInvoker, // 实际的gRPC调用函数
		grpcOpts ...grpc.CallOption, // gRPC调用选项
	) error {
		// 检查熔断器是否允许执行本次请求
		// 如果返回false，说明当前负载过高，需要节流
		if !breaker.Allow() {
			// 返回资源耗尽错误，指示客户端应该重试或降级处理
			return status.Error(codes.ResourceExhausted, "circuitbreaker: adaptive throttling, request dropped")
		}

		// 执行实际的gRPC调用
		err := invoker(ctx, method, req, reply, cc, grpcOpts...)

		if err == nil {
			breaker.MarkSuccess()
			return nil
		}

		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.DeadlineExceeded,
				codes.Internal,
				codes.Unavailable,
				codes.ResourceExhausted:
				breaker.MarkFailure()
			default:
				breaker.MarkSuccess()
			}
		} else {
			breaker.MarkFailure()
		}

		return err
	}
}
