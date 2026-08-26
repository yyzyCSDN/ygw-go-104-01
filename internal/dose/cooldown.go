package dose

import (
	"sync"
	"time"
)

type Cooldown struct {
	mu       sync.Mutex
	interval time.Duration
	last     map[string]time.Time
}

func NewCooldown(interval time.Duration) *Cooldown {
	return &Cooldown{
		interval: interval,
		last:     make(map[string]time.Time),
	}
}

func (c *Cooldown) Allow(source string, now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	last, ok := c.last[source]
	if !ok {
		return true
	}
	return now.Sub(last) >= c.interval
}

// Reset 清除指定来源的冷却记录,用于值班员手动重新下发启动时绕过自动冷却。
func (c *Cooldown) Reset(source string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.last, source)
}

func (c *Cooldown) Mark(source string, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.last[source] = now
}
