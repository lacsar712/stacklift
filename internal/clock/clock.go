package clock

import (
	"sync"
	"sync/atomic"
	"time"
)

type ProcessClock struct {
	mu        sync.Mutex
	tick      int64
	stepMS    int64
	paused    bool
	started   time.Time
	listeners []func(int64)
}

func New(stepMS int64) *ProcessClock {
	if stepMS <= 0 {
		stepMS = 100
	}
	return &ProcessClock{tick: 0, stepMS: stepMS, started: time.Now()}
}

func (c *ProcessClock) Tick() int64 { return atomic.LoadInt64(&c.tick) }

func (c *ProcessClock) Advance(n int64) int64 {
	if n <= 0 {
		return c.Tick()
	}
	c.mu.Lock()
	if c.paused {
		cur := c.tick
		c.mu.Unlock()
		return cur
	}
	c.tick += n
	cur := c.tick
	ls := append([]func(int64){}, c.listeners...)
	c.mu.Unlock()
	atomic.StoreInt64(&c.tick, cur)
	for _, fn := range ls {
		fn(cur)
	}
	return cur
}

func (c *ProcessClock) AdvanceOne() int64 { return c.Advance(1) }

func (c *ProcessClock) Pause() {
	c.mu.Lock()
	c.paused = true
	c.mu.Unlock()
}

func (c *ProcessClock) Paused() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.paused
}

func (c *ProcessClock) ElapsedTicks(since int64) int64 {
	cur := c.Tick()
	if cur < since {
		return 0
	}
	return cur - since
}

func (c *ProcessClock) WindowClosed(startTick, windowTicks int64) bool {
	if windowTicks <= 0 {
		return true
	}
	return c.ElapsedTicks(startTick) >= windowTicks
}
