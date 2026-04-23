package limiter

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
)

var (
	cpuUsage    atomic.Value
	cpuInterval int64 = int64(time.Millisecond * 500)
	collectOnce sync.Once
)

func init() {
	cpuUsage.Store(0.0)
}

func defaultCPU() float64 {
	collectOnce.Do(func() { go collectCPU() })
	return cpuUsage.Load().(float64)
}

func setCPUInterval(interval time.Duration) {
	if interval > 0 {
		atomic.StoreInt64(&cpuInterval, int64(interval))
	}
}

func collectCPU() {
	for {
		interval := time.Duration(atomic.LoadInt64(&cpuInterval))
		percentages, err := cpu.Percent(interval, false)
		if err == nil && len(percentages) > 0 {
			cpuUsage.Store(percentages[0] / 100.0)
		} else {
			time.Sleep(interval)
		}
	}
}
