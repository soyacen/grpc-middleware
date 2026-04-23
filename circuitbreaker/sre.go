// Package circuitbreaker 提供gRPC客户端熔断器中间件功能
// 实现基于Google SRE自适应节流算法的gRPC客户端拦截器
package circuitbreaker

import (
	"math/rand"
	"sync"
	"time"
)

var rndPool = sync.Pool{
	New: func() interface{} {
		return rand.New(rand.NewSource(time.Now().UnixNano()))
	},
}

// options 熔断器配置选项结构体
// 用于配置SRE熔断器的核心参数
type options struct {
	// K 熔断因子，用于计算熔断阈值的倍数系数
	// 当错误率超过正常错误率的 K 倍时触发熔断
	K float64

	// Window 统计时间窗口，用于计算错误率的时间范围
	// 单位为时间 duration
	Window time.Duration

	// Buckets 时间窗口内的桶数量
	// 用于将时间窗口分割成多个小的时间段进行统计
	Buckets int
}

// Option 配置选项函数类型
// 用于链式调用设置各种配置参数
type Option func(*options)

// WithK 设置熔断因子 K 值
// 参数 k: 熔断倍数因子，建议值范围 1.5-3.0
func WithK(k float64) Option {
	return func(o *options) {
		o.K = k
	}
}

// WithWindow 设置统计时间窗口
// 参数 window: 统计错误率的时间窗口长度
func WithWindow(window time.Duration) Option {
	return func(o *options) {
		o.Window = window
	}
}

// WithBuckets 设置时间窗口内的桶数量
// 参数 buckets: 将时间窗口分割的桶数量，影响统计精度
func WithBuckets(buckets int) Option {
	return func(o *options) {
		o.Buckets = buckets
	}
}

// defaultOptions 返回默认的熔断器配置选项
// 默认值：
// - K: 2.0 (错误率倍数因子)
// - Window: 10秒 (统计时间窗口)
// - Buckets: 40 (时间桶数量)
func defaultOptions() *options {
	return &options{
		K:       2.0,
		Window:  time.Second * 10,
		Buckets: 40,
	}
}

// init 初始化配置选项，确保所有参数都有有效值
// 如果参数未设置或设置为无效值，则使用默认值
func (o *options) init() *options {
	if o.K <= 0 {
		o.K = 2.0
	}
	if o.Window <= 0 {
		o.Window = time.Second * 10
	}
	if o.Buckets <= 0 {
		o.Buckets = 40
	}
	return o
}

// apply 应用所有配置选项到设置中
//
// 参数:
//   - st: 目标设置
//   - opts: 配置选项列表
func (o *options) apply(opts ...Option) *options {
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func (o *options) newSREBreaker() SREBreaker {
	return &sreBreaker{
		k:      o.K,
		window: newWindow(o.Window, o.Buckets),
		rnd:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// SREBreaker 实现Google SRE手册中的自适应节流算法
// 参考: https://sre.google/sre-book/handling-overload/#eq2101
// 该接口定义了熔断器的核心行为
type SREBreaker interface {
	// Allow 判断是否允许执行请求
	// 返回true表示允许执行，false表示应该拒绝请求
	Allow() bool

	// MarkSuccess 标记一次成功的请求
	// 在请求成功完成后调用此方法
	MarkSuccess()

	// MarkFailure 标记一次失败的请求
	// 在请求失败后调用此方法
	MarkFailure()
}

// sreBreaker SRE熔断器的具体实现
type sreBreaker struct {
	k          float64
	window     *window
	rndMu      sync.Mutex
	rnd        *rand.Rand
}

func (b *sreBreaker) Allow() bool {
	requests, accepts := b.window.Summary()

	p := (float64(requests) - b.k*float64(accepts)) / float64(requests+1)

	if p <= 0 {
		return true
	}

	b.rndMu.Lock()
	defer b.rndMu.Unlock()
	return b.rnd.Float64() >= p
}

// MarkSuccess 标记一次成功的请求
// 增加成功计数和总请求数
func (b *sreBreaker) MarkSuccess() {
	b.window.Add(1, 1)
}

// MarkFailure 标记一次失败的请求
// 只增加总请求数，不增加成功计数
func (b *sreBreaker) MarkFailure() {
	b.window.Add(1, 0)
}

// window 滑动时间窗口结构体
// 用于在固定时间范围内统计请求数据
type window struct {
	// mu 读写锁，保证并发安全
	mu sync.RWMutex

	// buckets 时间桶数组，每个桶存储一段时间内的统计数据
	buckets []bucket

	// windowSize 总时间窗口大小
	windowSize time.Duration

	// bucketSize 每个时间桶的大小
	bucketSize time.Duration

	// lastUpdate 上次更新时间，用于桶轮转
	lastUpdate time.Time
}

// bucket 时间桶结构体
// 存储单个时间桶内的请求统计信息
type bucket struct {
	// requests 该时间段内的总请求数
	requests int64

	// accepts 该时间段内的成功请求数
	accepts int64
}

// newWindow 创建新的滑动窗口实例
// 参数:
//   - windowSize: 总时间窗口大小
//   - buckets: 时间桶数量
func newWindow(windowSize time.Duration, buckets int) *window {
	return &window{
		buckets:    make([]bucket, buckets),
		windowSize: windowSize,
		bucketSize: windowSize / time.Duration(buckets),
		lastUpdate: time.Now(),
	}
}

// Add 向窗口中添加统计数据
// 参数:
//   - requests: 新增的请求数
//   - accepts: 新增的成功请求数
func (w *window) Add(requests, accepts int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 执行桶轮转，清理过期数据
	w.rotate()

	// 计算当前时间对应的桶索引
	idx := (time.Now().UnixNano() / int64(w.bucketSize)) % int64(len(w.buckets))

	// 更新对应桶的统计数据
	w.buckets[idx].requests += requests
	w.buckets[idx].accepts += accepts
}

// Summary 获取窗口内所有桶的统计汇总
// 返回:
//   - requests: 总请求数
//   - accepts: 总成功请求数
func (w *window) Summary() (requests, accepts int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.rotate()

	for i := range w.buckets {
		requests += w.buckets[i].requests
		accepts += w.buckets[i].accepts
	}
	return
}

// rotate 执行桶轮转操作
// 清理超出时间窗口的旧数据桶
func (w *window) rotate() {
	now := time.Now()

	// 如果距离上次更新不足一个桶的时间间隔，则无需轮转
	if now.Sub(w.lastUpdate) < w.bucketSize {
		return
	}

	// 计算需要清空的桶数量
	elapsed := now.Sub(w.lastUpdate)
	numToClear := int(elapsed / w.bucketSize)

	// 限制清空数量不超过桶总数
	if numToClear > len(w.buckets) {
		numToClear = len(w.buckets)
	}

	// 计算上次更新时的桶索引
	lastIdx := (w.lastUpdate.UnixNano() / int64(w.bucketSize)) % int64(len(w.buckets))

	// 清空相应数量的旧桶
	for i := 1; i <= numToClear; i++ {
		idx := (lastIdx + int64(i)) % int64(len(w.buckets))
		w.buckets[idx] = bucket{}
	}

	// 更新最后更新时间
	w.lastUpdate = now
}
