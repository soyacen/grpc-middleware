# grpc-middleware

[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://golang.org)
[![Go Reference](https://pkg.go.dev/badge/github.com/soyacen/grpc-middleware.svg)](https://pkg.go.dev/github.com/soyacen/grpc-middleware)

Go gRPC 中间件库，提供多种生产级拦截器组件，包括限流、熔断、认证、日志、重试、超时等功能。

## 功能特性

| 中间件 | 类型 | 说明 |
|--------|------|------|
| **ratelimiter** | Server | BBR 自适应限流，支持 CPU 过载保护 |
| **circuitbreaker** | Client | Google SRE 熔断算法 |
| **auth** | Server | 认证元数据处理 |
| **accesslog** | Server/Client | 访问日志记录 |
| **retry** | Client | 指数退避重试机制 |
| **timeout** | Client | 请求超时控制 |
| **errorlog** | Server/Client | 错误日志记录 |
| **slowlog** | Server/Client | 慢请求检测与日志 |
| **recovery** | Server | Panic 恢复处理 |
| **context** | Server/Client | 上下文修改与传递 |
| **unifiederror** | Server | 统一错误响应格式 |

## 安装

```bash
go get github.com/soyacen/grpc-middleware
```

## 快速开始

### 服务端使用示例

```go
package main

import (
    "google.golang.org/grpc"
    "github.com/soyacen/grpc-middleware/ratelimiter"
    "github.com/soyacen/grpc-middleware/auth"
    "github.com/soyacen/grpc-middleware/recovery"
    "github.com/soyacen/grpc-middleware/accesslog"
)

func main() {
    // 创建服务器拦截器链
    server := grpc.NewServer(
        grpc.ChainUnaryInterceptor(
            // 限流：BBR 自适应限流，CPU 阈值 80%
            ratelimiter.UnaryServerInterceptor(
                ratelimiter.WithCPUThreshold(0.8),
            ),
            // 认证：验证 JWT Token
            auth.UnaryServerInterceptor(func(ctx context.Context, fullMethod string) (context.Context, error) {
                // 认证逻辑
                return ctx, nil
            }),
            // 访问日志
            accesslog.UnaryServerInterceptor(),
            // Panic 恢复
            recovery.UnaryServerInterceptor(),
        ),
    )
    
    // 注册服务并启动...
}
```

### 客户端使用示例

```go
package main

import (
    "google.golang.org/grpc"
    "github.com/soyacen/grpc-middleware/circuitbreaker"
    "github.com/soyacen/grpc-middleware/retry"
    "github.com/soyacen/grpc-middleware/timeout"
)

func main() {
    conn, err := grpc.Dial(
        "localhost:50051",
        grpc.WithChainUnaryInterceptor(
            // 熔断：基于 Google SRE 算法
            circuitbreaker.UnaryClientInterceptor(),
            // 超时：3秒超时
            timeout.UnaryClientInterceptor(
                timeout.WithTimeout(3 * time.Second),
            ),
            // 重试：最多3次，指数退避
            retry.UnaryClientInterceptor(
                retry.WithMaxAttempts(3),
                retry.WithBackoff(time.Second, 5 * time.Second),
            ),
        ),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()
    
    // 使用连接...
}
```

## 中间件分类

### 服务端中间件

适用于 gRPC 服务端，处理进入的请求：

- **ratelimiter** - BBR 自适应限流，防止服务过载
- **auth** - 请求认证处理
- **recovery** - Panic 捕获与恢复
- **accesslog** - 完整的访问日志记录
- **errorlog** - 仅记录错误请求
- **slowlog** - 慢请求检测（可配置阈值）
- **unifiederror** - 统一错误响应格式
- **context** - 上下文修改与注入

### 客户端中间件

适用于 gRPC 客户端，处理发出的请求：

- **circuitbreaker** - 熔断保护，防止级联故障
- **retry** - 自动重试失败请求
- **timeout** - 请求超时控制
- **accesslog** - 客户端调用日志
- **errorlog** - 客户端错误日志
- **slowlog** - 客户端慢请求检测
- **context** - 客户端上下文修改

## 项目结构

```
.
├── ratelimiter/      # BBR 限流算法 + CPU 监控
├── circuitbreaker/   # SRE 熔断算法
├── auth/             # 认证中间件
├── accesslog/        # 访问日志（服务端/客户端）
├── errorlog/         # 错误日志
├── slowlog/          # 慢请求日志
├── retry/            # 客户端重试
├── timeout/          # 客户端超时
├── context/          # 上下文处理
├── recovery/         # Panic 恢复
├── unifiederror/     # 统一错误处理
└── doc.go            # 根包声明
```

## 配置选项

每个中间件都使用函数式选项模式（Functional Options Pattern）：

```go
// 限流器配置示例
ratelimiter.UnaryServerInterceptor(
    ratelimiter.WithWindow(10 * time.Second),      // 统计窗口
    ratelimiter.WithBuckets(100),                   // 桶数量
    ratelimiter.WithCPUThreshold(0.8),              // CPU 阈值
    ratelimiter.WithSkip(func() bool {             // 跳过条件
        return someCondition
    }),
)
```

## 测试

```bash
# 运行所有测试
go test ./...

# 带覆盖率测试
go test -coverprofile=coverage.out ./...

# 查看覆盖率报告
go tool cover -html=coverage.out
```

## 依赖

- `google.golang.org/grpc` - gRPC 核心库
- `github.com/shirou/gopsutil/v4` - 系统/CPU 监控（限流器使用）
- `github.com/soyacen/gox` - 作者工具库

## 许可证

MIT License
