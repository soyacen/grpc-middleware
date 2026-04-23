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

type sreBreaker struct {
	k      float64
	window *window
	rndMu  sync.Mutex
	rnd    *rand.Rand
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

func (b *sreBreaker) MarkSuccess() {
	b.window.Add(1, 1)
}

func (b *sreBreaker) MarkFailure() {
	b.window.Add(1, 0)
}
