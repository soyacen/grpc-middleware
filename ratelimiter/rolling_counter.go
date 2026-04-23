package ratelimiter

import (
	"math"
	"sync"
	"time"
)

// rollingCounter 滚动计数器，用于在滑动窗口内统计数据
type rollingCounter struct {
	mu         sync.RWMutex
	buckets    []int64
	window     time.Duration
	bucketDur  time.Duration
	lastUpdate time.Time
	isMin      bool
}

// newRollingCounter 创建滚动计数器
// isMin为true时记录最小值，否则记录累加值
func newRollingCounter(window time.Duration, buckets int, isMin bool) *rollingCounter {
	rc := &rollingCounter{
		buckets:    make([]int64, buckets),
		window:     window,
		bucketDur:  window / time.Duration(buckets),
		lastUpdate: time.Now(),
		isMin:      isMin,
	}
	if isMin {
		for i := range rc.buckets {
			rc.buckets[i] = math.MaxInt64
		}
	}
	return rc
}

// Add 在指定时间添加数值
func (c *rollingCounter) Add(now time.Time, val int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.rotateAt(now)

	idx := (now.UnixNano() / int64(c.bucketDur)) % int64(len(c.buckets))
	if c.isMin {
		if val < c.buckets[idx] {
			c.buckets[idx] = val
		}
	} else {
		c.buckets[idx] += val
	}
}

// Max 返回所有桶中的最大值
func (c *rollingCounter) Max(now time.Time) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rotateAt(now)

	var max int64
	for _, v := range c.buckets {
		if !c.isMin {
			if v > max {
				max = v
			}
		} else {
			if v != math.MaxInt64 && v > max {
				max = v
			}
		}
	}
	return max
}

// Min 返回所有桶中的最小值
func (c *rollingCounter) Min(now time.Time) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rotateAt(now)

	var min int64 = math.MaxInt64
	var found bool
	for _, v := range c.buckets {
		if v > 0 && v != math.MaxInt64 {
			if v < min {
				min = v
			}
			found = true
		}
	}

	if !found {
		return 0
	}
	return min
}

// rotateAt 清理过期的桶
func (c *rollingCounter) rotateAt(now time.Time) {
	if now.Sub(c.lastUpdate) < c.bucketDur {
		return
	}

	elapsed := now.Sub(c.lastUpdate)
	numToClear := int(elapsed / c.bucketDur)
	if numToClear > len(c.buckets) {
		numToClear = len(c.buckets)
	}

	lastIdx := (c.lastUpdate.UnixNano() / int64(c.bucketDur)) % int64(len(c.buckets))
	for i := 1; i <= numToClear; i++ {
		idx := (lastIdx + int64(i)) % int64(len(c.buckets))
		if c.isMin {
			c.buckets[idx] = math.MaxInt64
		} else {
			c.buckets[idx] = 0
		}
	}

	c.lastUpdate = now
}
