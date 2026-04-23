package container

import (
	"math"
	"sync"
	"time"
)

type RollingCounter struct {
	mu         sync.RWMutex
	buckets    []int64
	window     time.Duration
	bucketDur  time.Duration
	lastUpdate time.Time
	isMin      bool
}

func NewRollingCounter(window time.Duration, buckets int, isMin bool) *RollingCounter {
	rc := &RollingCounter{
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

func (c *RollingCounter) Add(now time.Time, val int64) {
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

func (c *RollingCounter) rotateAt(now time.Time) {
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

func (c *RollingCounter) Max(now time.Time) int64 {
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

func (c *RollingCounter) Min(now time.Time) int64 {
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
