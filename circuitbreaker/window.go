package circuitbreaker

import (
	"sync"
	"time"
)

type bucket struct {
	requests int64
	accepts  int64
}

type window struct {
	mu         sync.RWMutex
	buckets    []bucket
	windowSize time.Duration
	bucketSize time.Duration
	lastUpdate time.Time
}

func newWindow(windowSize time.Duration, buckets int) *window {
	return &window{
		buckets:    make([]bucket, buckets),
		windowSize: windowSize,
		bucketSize: windowSize / time.Duration(buckets),
		lastUpdate: time.Now(),
	}
}

func (w *window) Add(requests, accepts int64) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.rotate()

	idx := (time.Now().UnixNano() / int64(w.bucketSize)) % int64(len(w.buckets))
	w.buckets[idx].requests += requests
	w.buckets[idx].accepts += accepts
}

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

func (w *window) rotate() {
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
