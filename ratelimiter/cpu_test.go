package ratelimiter

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultCPU(t *testing.T) {
	usage := defaultCPU()
	assert.GreaterOrEqual(t, usage, 0.0)
	assert.LessOrEqual(t, usage, 1.0)
}

func TestSetCPUInterval(t *testing.T) {
	original := atomic.LoadInt64(&cpuInterval)
	defer atomic.StoreInt64(&cpuInterval, original)

	setCPUInterval(time.Millisecond * 200)
	assert.Equal(t, int64(time.Millisecond*200), atomic.LoadInt64(&cpuInterval))
}

func TestSetCPUInterval_ZeroIgnored(t *testing.T) {
	original := atomic.LoadInt64(&cpuInterval)
	defer atomic.StoreInt64(&cpuInterval, original)

	setCPUInterval(time.Millisecond * 200)
	setCPUInterval(0)

	assert.Equal(t, int64(time.Millisecond*200), atomic.LoadInt64(&cpuInterval))
}

func TestCPUInterval_Atomic(t *testing.T) {
	original := atomic.LoadInt64(&cpuInterval)
	defer atomic.StoreInt64(&cpuInterval, original)

	setCPUInterval(time.Millisecond * 100)
	val := atomic.LoadInt64(&cpuInterval)
	assert.Equal(t, int64(time.Millisecond*100), val)

	setCPUInterval(time.Second)
	val = atomic.LoadInt64(&cpuInterval)
	assert.Equal(t, int64(time.Second), val)
}

func TestCPU_WithCPU_Injection(t *testing.T) {
	customCPU := func() float64 { return 0.5 }
	opts := defaultOptions().apply(WithCPU(customCPU)).init()

	assert.NotNil(t, opts.CPU)
	assert.Equal(t, 0.5, opts.CPU())
}

func TestCPU_InitSetsInterval(t *testing.T) {
	original := atomic.LoadInt64(&cpuInterval)
	defer atomic.StoreInt64(&cpuInterval, original)

	opts := defaultOptions().apply(WithCPUInterval(time.Millisecond * 300)).init()
	assert.Equal(t, time.Millisecond*300, opts.CPUInterval)
}

func TestCPU_DefaultValue(t *testing.T) {
	usage := cpuUsage.Load()
	assert.NotNil(t, usage)

	val, ok := usage.(float64)
	assert.True(t, ok)
	assert.GreaterOrEqual(t, val, 0.0)
	assert.LessOrEqual(t, val, 1.0)
}
