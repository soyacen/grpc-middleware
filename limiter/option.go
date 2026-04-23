package limiter

import (
	"time"

	"github.com/soyacen/grpc-middleware/internal/container"
)

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
