package circuitbreaker

import (
	"math/rand/v2"
	"sync"

	"github.com/soyacen/gox/randx"
)

// rndPool 随机数生成器对象池
var rndPool = sync.Pool{
	New: func() interface{} {
		r, err := randx.NewPCG() // Create a new PCG generator if none available.
		if err != nil {
			panic(err) // Panic on failure to initialize due to crypto/rand issues.
		}
		return r
	},
}

// sreCircuitBreaker SRE熔断器实现
// 参考: https://sre.google/sre-book/handling-overload/#eq2101
type sreCircuitBreaker struct {
	k      float64
	window *rollingCounter
	rndMu  sync.Mutex
	rnd    *rand.Rand
}

// Allow 判断请求是否被允许
func (b *sreCircuitBreaker) Allow() bool {
	requests, accepts := b.window.Summary()

	p := (float64(requests) - b.k*float64(accepts)) / float64(requests+1)

	if p <= 0 {
		return true
	}

	b.rndMu.Lock()
	defer b.rndMu.Unlock()
	return b.rnd.Float64() >= p
}

// MarkSuccess 标记请求成功
func (b *sreCircuitBreaker) MarkSuccess() {
	b.window.Add(1, 1)
}

// MarkFailure 标记请求失败
func (b *sreCircuitBreaker) MarkFailure() {
	b.window.Add(1, 0)
}
