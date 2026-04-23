package ratelimiter

import (
	"sync/atomic"
	"time"
)

// bbrRateLimiter BBR限流器实现
type bbrRateLimiter struct {
	// conf 配置选项
	conf *options

	// inflight 当前并发请求数
	inflight int64

	// passStat 成功请求数统计
	passStat *rollingCounter

	// rtStat 响应时间统计
	rtStat *rollingCounter

	// lastDrop 上次丢弃请求的时间
	lastDrop atomic.Pointer[time.Time]

	// cpu CPU使用率获取函数
	cpu func() float64
}

// Allow 检查是否允许执行请求
func (l *bbrRateLimiter) Allow() (func(DoneInfo), error) {
	if l.shouldDrop() {
		return nil, ErrLimitExceeded
	}

	// 增加并发计数
	atomic.AddInt64(&l.inflight, 1)
	startTime := time.Now()

	// 返回完成回调函数
	return func(info DoneInfo) {
		// 确保减少并发计数
		defer atomic.AddInt64(&l.inflight, -1)

		// 计算响应时间（毫秒）
		rt := int64(time.Since(startTime) / time.Millisecond)

		// 更新统计信息
		if info.Err == nil {
			now := time.Now()
			l.passStat.Add(now, 1)
			l.rtStat.Add(now, rt)
		}
	}, nil
}

// shouldDrop 判断是否应丢弃请求
func (l *bbrRateLimiter) shouldDrop() bool {
	// 检查CPU使用率
	cpu := l.cpu()
	if cpu < l.conf.CPUThreshold {
		lastDrop := l.lastDrop.Load()
		if lastDrop == nil {
			return false
		}
		if time.Since(*lastDrop) > time.Second {
			l.lastDrop.Store(nil)
			return false
		}
	}

	// 检查并发数
	inflight := atomic.LoadInt64(&l.inflight)
	if inflight <= 1 {
		return false
	}

	// 检查是否超过最大并发数
	if float64(inflight) > l.maxInflight() {
		now := time.Now()
		l.lastDrop.Store(&now)
		return true
	}

	return false
}

// maxInflight 计算最大允许并发数
// BBR算法: max_inflight = max_pass * min_rt / (bucket_duration * 1000)
func (l *bbrRateLimiter) maxInflight() float64 {
	now := time.Now()
	maxPass := l.passStat.Max(now)
	minRT := l.rtStat.Min(now)

	if maxPass <= 0 || minRT <= 0 {
		return float64(l.conf.Buckets)
	}

	bucketDuration := float64(l.conf.Window) / float64(l.conf.Buckets) / float64(time.Second)
	return float64(maxPass) * float64(minRT) / 1000.0 / bucketDuration
}
