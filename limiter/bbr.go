package limiter

import (
	"sync/atomic"
	"time"

	"github.com/soyacen/grpc-middleware/internal/container"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrLimitExceeded 当限流触发时返回的错误
// 表示请求被限流器拒绝
var ErrLimitExceeded = status.Error(codes.ResourceExhausted, "limiter: rate limit exceeded")

// DoneInfo 包含请求执行完成的信息
type DoneInfo struct {
	// Err 是请求处理器返回的错误
	Err error
}

// Limiter 定义限流器接口
type Limiter interface {
	// Allow 检查请求是否被允许
	// 返回完成回调函数和可能的错误
	// 如果请求不被允许，返回错误
	Allow() (done func(DoneInfo), err error)
}

// bbrLimiter BBR限流器的具体实现
// 基于字节跳动BBR算法实现自适应限流
type bbrLimiter struct {
	// conf 配置选项引用
	conf *options

	// inflight 当前正在处理的请求数
	inflight int64

	// metrics for the BBR algorithm
	passStat *container.RollingCounter // 通过请求数统计
	rtStat   *container.RollingCounter // 响应时间统计

	// lastDrop 上次丢弃请求的时间
	lastDrop atomic.Pointer[time.Time]

	// cpu 获取当前CPU使用率的函数
	cpu func() float64
}

// Allow 检查请求是否允许执行
// 返回完成回调函数和可能的错误
func (l *bbrLimiter) Allow() (func(DoneInfo), error) {
	if l.shouldDrop() {
		return nil, ErrLimitExceeded
	}

	// 增加当前并发请求数
	atomic.AddInt64(&l.inflight, 1)
	startTime := time.Now()

	// 返回完成回调函数
	return func(info DoneInfo) {
		// 确保减少并发请求数，即使发生了异常
		defer atomic.AddInt64(&l.inflight, -1)

		// 计算请求处理时间(毫秒)
		rt := int64(time.Since(startTime) / time.Millisecond)

		// 如果请求成功，更新统计信息
		if info.Err == nil {
			now := time.Now()
			l.passStat.Add(now, 1) // 增加成功请求数
			l.rtStat.Add(now, rt)  // 记录响应时间
		}
	}, nil
}

// shouldDrop 判断是否应该丢弃请求
// 实现BBR算法的核心决策逻辑
func (l *bbrLimiter) shouldDrop() bool {
	// 检查CPU使用率
	cpu := l.cpu()
	if cpu < l.conf.CPUThreshold {
		// CPU使用率低于阈值
		lastDrop := l.lastDrop.Load()
		if lastDrop == nil {
			// 没有最近的丢弃记录，允许请求
			return false
		}
		// 检查距离上次丢弃是否已经超过1秒
		if time.Since(*lastDrop) > time.Second {
			// 超过1秒，清除丢弃记录，允许请求
			l.lastDrop.Store(nil)
			return false
		}
	}

	// 检查并发请求数
	inflight := atomic.LoadInt64(&l.inflight)
	if inflight <= 1 {
		// 并发数很少，允许请求
		return false
	}

	// 检查是否超过最大允许并发数
	if float64(inflight) > l.maxInflight() {
		// 超过限制，记录丢弃时间并拒绝请求
		now := time.Now()
		l.lastDrop.Store(&now)
		return true
	}

	return false
}

// maxInflight 计算最大允许并发请求数
// 基于BBR算法：max_inflight = max_pass * min_rt
func (l *bbrLimiter) maxInflight() float64 {
	now := time.Now()
	maxPass := l.passStat.Max(now)
	minRT := l.rtStat.Min(now)

	if maxPass <= 0 || minRT <= 0 {
		return float64(l.conf.Buckets)
	}

	bucketDuration := float64(l.conf.Window) / float64(l.conf.Buckets) / float64(time.Second)
	return float64(maxPass) * float64(minRT) / 1000.0 / bucketDuration
}
