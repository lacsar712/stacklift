// Package counter tracks hoist cycle counts with cross-package error sentinels.
package counter

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrCycleLimit   = errors.New("counter: cycle limit")
	ErrCounterStale = errors.New("counter: stale counter")
)

type Fault struct {
	RigID string
	Cause error
	At    time.Time
}

func (f *Fault) Error() string {
	if f == nil {
		return "counter: nil fault"
	}
	return fmt.Sprintf("counter fault rig=%s: %v", f.RigID, f.Cause)
}
func (f *Fault) Unwrap() error {
	if f == nil {
		return nil
	}
	return f.Cause
}

func Wrap(rigID string, cause error) error {
	if cause == nil {
		return nil
	}
	return &Fault{RigID: rigID, Cause: cause, At: time.Now().UTC()}
}

func IsLimit(err error) bool { return errors.Is(err, ErrCycleLimit) }
func IsStale(err error) bool { return errors.Is(err, ErrCounterStale) }

type CycleCounter struct {
	mu         sync.Mutex
	counts     map[string]int
	maxCycles  int
	updated    map[string]time.Time
	staleAfter time.Duration
}

func New(maxCycles int) *CycleCounter {
	return &CycleCounter{
		counts: make(map[string]int), maxCycles: maxCycles,
		updated: make(map[string]time.Time), staleAfter: 5 * time.Minute,
	}
}

func (c *CycleCounter) Increment(rigID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counts[rigID]++
	c.updated[rigID] = time.Now().UTC()
	if c.maxCycles > 0 && c.counts[rigID] > c.maxCycles {
		return Wrap(rigID, ErrCycleLimit)
	}
	return nil
}

func (c *CycleCounter) Count(rigID string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.counts[rigID]
}

func (c *CycleCounter) Reset(rigID string) {
	c.mu.Lock()
	c.counts[rigID] = 0
	c.updated[rigID] = time.Now().UTC()
	c.mu.Unlock()
}

func (c *CycleCounter) Check(rigID string, now time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	t, ok := c.updated[rigID]
	if !ok {
		return Wrap(rigID, ErrCounterStale)
	}
	if now.Sub(t) > c.staleAfter {
		return Wrap(rigID, ErrCounterStale)
	}
	if c.maxCycles > 0 && c.counts[rigID] > c.maxCycles {
		return Wrap(rigID, ErrCycleLimit)
	}
	return nil
}

func (c *CycleCounter) Snapshot() map[string]int {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[string]int, len(c.counts))
	for id, n := range c.counts {
		out[id] = n
	}
	return out
}
