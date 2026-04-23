package container

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRollingCounter(t *testing.T) {
	t.Run("basic_add_and_max", func(t *testing.T) {
		c := NewRollingCounter(time.Millisecond*100, 10, false)
		c.Add(time.Now(), 10)
		assert.Equal(t, int64(10), c.Max(time.Now()))
	})

	t.Run("rotation_expires_old_data", func(t *testing.T) {
		c := NewRollingCounter(time.Millisecond*100, 10, false)
		c.Add(time.Now(), 10)
		time.Sleep(time.Millisecond * 150)
		c.Add(time.Now(), 5)
		assert.Equal(t, int64(5), c.Max(time.Now()))
	})

	t.Run("min_tracking", func(t *testing.T) {
		c := NewRollingCounter(time.Millisecond*100, 10, true)
		c.Add(time.Now(), 100)
		c.Add(time.Now(), 50)
		c.Add(time.Now(), 200)
		assert.Equal(t, int64(50), c.Min(time.Now()))
	})

	t.Run("min_returns_zero_when_empty", func(t *testing.T) {
		c := NewRollingCounter(time.Second, 10, true)
		assert.Equal(t, int64(0), c.Min(time.Now()))
	})

	t.Run("max_returns_zero_when_empty", func(t *testing.T) {
		c := NewRollingCounter(time.Second, 10, false)
		assert.Equal(t, int64(0), c.Max(time.Now()))
	})

	t.Run("stale_data_cleared_on_read", func(t *testing.T) {
		c := NewRollingCounter(time.Millisecond*50, 5, false)
		c.Add(time.Now(), 100)
		time.Sleep(time.Millisecond * 300)
		assert.Equal(t, int64(0), c.Max(time.Now()))
	})

	t.Run("concurrent_add", func(t *testing.T) {
		c := NewRollingCounter(time.Second, 100, false)
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				c.Add(time.Now(), 1)
			}()
		}
		wg.Wait()
		assert.GreaterOrEqual(t, c.Max(time.Now()), int64(1))
	})
}
