package telemetry

import "sync"

type Counters struct {
	mu    sync.Mutex
	count map[string]int64
}

func NewCounters() *Counters { return &Counters{count: make(map[string]int64)} }

func (c *Counters) Inc(name string, delta int64) {
	c.mu.Lock()
	c.count[name] += delta
	c.mu.Unlock()
}

func (c *Counters) Snapshot() map[string]int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[string]int64, len(c.count))
	for k, v := range c.count {
		out[k] = v
	}
	return out
}
