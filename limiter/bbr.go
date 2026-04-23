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

// options 限流器配置选项结构体
// 用于配置BBR限流算法的各项参数
type options struct {
	// Window 统计时间窗口
	// 用于计算请求通过率和响应时间的时间范围
	Window time.Duration

	// Buckets 时间窗口内的桶数量
	// 将时间窗口分割成多个桶进行精细化统计
	Buckets int

	// CPUThreshold CPU触发阈值 (0.0-1.0)
	// 当CPU使用率超过此阈值时开始限流
	// 单位为小数，例如0.8表示80%
	CPUThreshold float64

	// CPU 获取当前CPU使用率的函数
	// 返回值范围0.0-1.0
	CPU func() float64

	// CPUInterval CPU采样间隔
	CPUInterval time.Duration

	// Skip 跳过限流的判断函数
	// 返回true时表示跳过限流检查，直接允许请求
	Skip func() bool
}

// Option 配置选项函数类型
// 用于链式调用设置各种配置参数
type Option func(*options)

// WithWindow 设置统计时间窗口
// 参数 window: 统计时间窗口长度
func WithWindow(window time.Duration) Option {
	return func(o *options) {
		o.Window = window
	}
}

// WithBuckets 设置时间窗口内的桶数量
// 参数 buckets: 时间桶数量，影响统计精度
func WithBuckets(buckets int) Option {
	return func(o *options) {
		o.Buckets = buckets
	}
}

// WithCPUThreshold 设置CPU触发阈值 (0.0-1.0)
// 参数 threshold: CPU使用率阈值，超过此值开始限流
func WithCPUThreshold(threshold float64) Option {
	return func(o *options) {
		o.CPUThreshold = threshold
	}
}

// WithCPU 设置CPU获取函数
// 参数 cpu: 自定义的CPU使用率获取函数
func WithCPU(cpu func() float64) Option {
	return func(o *options) {
		o.CPU = cpu
	}
}

// WithCPUInterval 设置CPU采样间隔
// 参数 interval: CPU采样的时间间隔
func WithCPUInterval(interval time.Duration) Option {
	return func(o *options) {
		o.CPUInterval = interval
	}
}

// WithSkip 设置跳过限流的判断函数
// 参数 skip: 返回true时跳过限流检查，直接允许请求
func WithSkip(skip func() bool) Option {
	return func(o *options) {
		o.Skip = skip
	}
}

// defaultOptions 返回默认配置
// 默认值：
// - Window: 10秒
// - Buckets: 100个
// - CPUThreshold: 0.8 (80%)
// - CPU: defaultCPU函数
func defaultOptions() *options {
	return &options{
		Window:       time.Second * 10,
		Buckets:      100,
		CPUThreshold: 0.8,
		CPU:          defaultCPU,
		CPUInterval:  time.Millisecond * 500,
	}
}

// init 初始化配置，确保所有参数有效
// 对未设置或无效的参数使用默认值
func (o *options) init() *options {
	if o.Window <= 0 {
		o.Window = time.Second * 10
	}
	if o.Buckets <= 0 {
		o.Buckets = 100
	}
	if o.CPUThreshold <= 0 {
		o.CPUThreshold = 0.8
	}
	if o.CPUInterval <= 0 {
		o.CPUInterval = time.Millisecond * 500
	}
	if o.CPU == nil {
		o.CPU = defaultCPU
		setCPUInterval(o.CPUInterval)
	}
	return o
}

// apply 应用配置选项
// 按顺序应用所有配置选项函数
func (o *options) apply(opts ...Option) *options {
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// newLimiter 创建新的限流器实例
// 根据配置选项初始化BBR限流器
func (o *options) newLimiter() Limiter {
	return &bbrLimiter{
		conf:     o,
		passStat: container.NewRollingCounter(o.Window, o.Buckets, false),
		rtStat:   container.NewRollingCounter(o.Window, o.Buckets, true),
		cpu:      o.CPU,
	}
}

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
