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
		// CPU 负载正常
		lastDrop := l.lastDrop.Load()
		if lastDrop == nil {
			// 从未丢弃过，放行
			return false
		}
		if time.Since(*lastDrop) > time.Second {
			// 冷却期已过（>1s），放行
			l.lastDrop.Store(nil)
			return false
		}
		// 即使 CPU 降到了阈值以下，如果在 1 秒内 刚丢弃过请求，仍然继续限流检查。
		// 为什么？ 防止 CPU 在阈值附近反复横跳导致请求抖动（thrashing）。一旦进入丢弃状态，至少持续检查 1 秒。
	}
	// CPU 超载 或 仍在冷却期内，继续检查
	// 检查并发数
	inflight := atomic.LoadInt64(&l.inflight)
	if inflight <= 1 {
		// 只有 1 个请求在跑，直接放行
		// 为什么？ 在流量极低时，统计数据（passStat、rtStat）可能不足以计算准确的承载能力。保留至少 1 个请求通过，
		// 避免冷启动误杀。
		return false
	}

	// 检查是否超过最大并发数， BBR 算法核心
	// 调用 maxInflight() 计算系统当前能承载的最大并发数
	if float64(inflight) > l.maxInflight() {
		// 当前并发数大于最大并发数，丢弃请求
		// 记录丢弃时间
		now := time.Now()
		l.lastDrop.Store(&now)
		// 丢弃请求
		return true
	}

	// 放行
	return false
}

// maxInflight 计算最大允许并发数
// BBR算法: max_inflight = max_pass * min_rt / (bucket_duration * 1000)
func (l *bbrRateLimiter) maxInflight() float64 {
	now := time.Now()
	// 窗口内最忙 bucket 的请求数
	maxPass := l.passStat.Max(now)
	// 窗口内的最小响应时间（ms）
	minRT := l.rtStat.Min(now)

	if maxPass <= 0 || minRT <= 0 {
		// 无数据时兜底
		return float64(l.conf.Buckets)
	}

	// BBR 思想：用历史最优吞吐（maxPass）乘以最小时延（minRT），估算系统的最佳并发窗口（类似 TCP BBR 的 BDP = bandwidth × RTT）。
	bucketDuration := float64(l.conf.Window) / float64(l.conf.Buckets) / float64(time.Second)
	return float64(maxPass) * float64(minRT) / 1000.0 / bucketDuration
}

//   ┌────────────────────────┬──────────┬──────────────────┐
//   │          场景           │   结果   │       原因        │
//   ├────────────────────────┼──────────┼──────────────────┤
//   │ CPU 正常 + 无丢弃记录    │ 放行      │ 系统健康          │
//   ├────────────────────────┼──────────┼──────────────────┤
//   │ CPU 正常 + 丢弃已过 1s   │ 放行     │ 冷却期结束         │
//   ├────────────────────────┼──────────┼──────────────────┤
//   │ CPU 正常 + 1s 内刚丢弃   │ 继续检查  │ 防抖动保护        │
//   ├────────────────────────┼──────────┼──────────────────┤
//   │ CPU 超载                │ 继续检查  │ 需进一步评估      │
//   ├────────────────────────┼──────────┼──────────────────┤
//   │ 并发 <= 1               │ 放行     │ 兜底保护          │
//   ├────────────────────────┼──────────┼──────────────────┤
//   │ 并发 > maxInflight      │ 丢弃     │ 超过系统承载能力   │
//   ├────────────────────────┼──────────┼──────────────────┤
//   │ 并发 <= maxInflight     │ 放行     │ 未超载            │
//   └────────────────────────┴──────────┴──────────────────┘
