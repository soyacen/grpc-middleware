package circuitbreaker

import (
	"sync"
	"time"
)

// bucket 统计桶
type bucket struct {
	requests int64
	accepts  int64
}

// rollingCounter 滑动时间窗口
type rollingCounter struct {
	mu         sync.RWMutex
	buckets    []bucket
	windowSize time.Duration
	bucketSize time.Duration
	lastUpdate time.Time
}

// newRollingCounter 创建滑动窗口
func newRollingCounter(windowSize time.Duration, buckets int) *rollingCounter {
	return &rollingCounter{
		buckets:    make([]bucket, buckets),
		windowSize: windowSize,
		bucketSize: windowSize / time.Duration(buckets),
		lastUpdate: time.Now(),
	}
}

// Add 添加统计数据
func (w *rollingCounter) Add(requests, accepts int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.rotate()

	idx := (time.Now().UnixNano() / int64(w.bucketSize)) % int64(len(w.buckets))
	w.buckets[idx].requests += requests
	w.buckets[idx].accepts += accepts
}

// Summary 获取统计汇总
func (w *rollingCounter) Summary() (requests, accepts int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.rotate()

	for i := range w.buckets {
		requests += w.buckets[i].requests
		accepts += w.buckets[i].accepts
	}
	return
}

// rotate 清理过期桶
func (w *rollingCounter) rotate() {
	now := time.Now()

	if now.Sub(w.lastUpdate) < w.bucketSize {
		return
	}

	elapsed := now.Sub(w.lastUpdate)
	numToClear := int(elapsed / w.bucketSize)

	if numToClear > len(w.buckets) {
		numToClear = len(w.buckets)
	}

	lastIdx := (w.lastUpdate.UnixNano() / int64(w.bucketSize)) % int64(len(w.buckets))

	for i := 1; i <= numToClear; i++ {
		idx := (lastIdx + int64(i)) % int64(len(w.buckets))
		w.buckets[idx] = bucket{}
	}

	w.lastUpdate = now
}
