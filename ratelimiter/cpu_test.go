package ratelimiter

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetCPUUsage(t *testing.T) {
	usage := defaultCPU()
	assert.GreaterOrEqual(t, usage, 0.0)
	assert.LessOrEqual(t, usage, 100.0)
}

func TestDefaultCPU(t *testing.T) {
	usage := defaultCPU()
	assert.GreaterOrEqual(t, usage, 0.0)
}

func TestCPUInterval(t *testing.T) {
	opts := defaultOptions().apply(WithCPUInterval(time.Millisecond * 100)).init()
	assert.Equal(t, time.Millisecond*100, opts.CPUInterval)
}

func TestSetCPUInterval(t *testing.T) {
	setCPUInterval(time.Millisecond * 200)
	assert.Equal(t, int64(time.Millisecond*200), atomic.LoadInt64(&cpuInterval))

	setCPUInterval(0)
	assert.Equal(t, int64(time.Millisecond*200), atomic.LoadInt64(&cpuInterval))
}
