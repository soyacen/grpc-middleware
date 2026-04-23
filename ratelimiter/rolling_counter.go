package ratelimiter

import (
	"math"
	"sync"
	"time"
)

// rollingCounter is a thread-safe sliding window counter that tracks values over a time window.
// It divides the window into fixed-size buckets and supports both sum (additive) and min modes.
// The counter automatically rotates buckets as time progresses.
type rollingCounter struct {
	mu         sync.RWMutex  // protects all fields
	buckets    []int64       // circular buffer of bucket values
	window     time.Duration // total time window for the counter
	bucketDur  time.Duration // duration of each bucket
	lastUpdate time.Time     // last time the counter was updated
	isMin      bool          // if true, tracks minimum values; otherwise tracks sums
}

// newRollingCounter creates a new RollingCounter.
// Parameters:
//   - window: total time window to track
//   - buckets: number of buckets to divide the window into
//   - isMin: if true, counter tracks minimum values; otherwise tracks sums
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

// Add adds a value to the counter at the given time.
// For sum mode (isMin=false), the value is added to the current bucket.
// For min mode (isMin=true), the minimum of the current value and new value is stored.
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

// Max returns the maximum value across all buckets at the given time.
// For min mode, only considers values that have been set (not MaxInt64).
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

// Min returns the minimum value across all buckets at the given time.
// Only considers values greater than 0 and not MaxInt64.
// Returns 0 if no valid values exist.
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

// rotateAt clears expired buckets based on the elapsed time since last update.
// Expired buckets are reset to 0 (for sum mode) or MaxInt64 (for min mode).
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
