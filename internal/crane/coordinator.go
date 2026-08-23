package crane

import (
	"context"
	"fmt"
	"sync"

	"github.com/lacsar712/stacklift/internal/clock"
	"github.com/lacsar712/stacklift/internal/model"
	"github.com/lacsar712/stacklift/internal/path"
)

type Coordinator struct {
	mu      sync.RWMutex
	cranes  map[model.CraneID]*Service
	planner *path.Planner
	clk     *clock.DualClock
	limits  model.LimitSet
}

func NewCoordinator(planner *path.Planner, clk *clock.DualClock, limits model.LimitSet) *Coordinator {
	return &Coordinator{cranes: make(map[model.CraneID]*Service), planner: planner, clk: clk, limits: limits}
}

func (c *Coordinator) Register(id model.CraneID) *Service {
	c.mu.Lock()
	defer c.mu.Unlock()
	svc := NewService(id, c.limits, c.clk)
	c.cranes[id] = svc
	return svc
}

func (c *Coordinator) Get(id model.CraneID) (*Service, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	svc, ok := c.cranes[id]
	return svc, ok
}

func (c *Coordinator) CraneIDs() []model.CraneID {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ids := make([]model.CraneID, 0, len(c.cranes))
	for id := range c.cranes {
		ids = append(ids, id)
	}
	return ids
}

func (c *Coordinator) MoveCrane(ctx context.Context, craneID model.CraneID, from, to model.Location) error {
	svc, ok := c.Get(craneID)
	if !ok {
		return model.ErrCraneNotFound
	}
	plan, err := c.planner.Plan("", craneID, from, to)
	if err != nil {
		return fmt.Errorf("plan move: %w", err)
	}
	return ExecutePlan(ctx, svc, plan)
}

func (c *Coordinator) CancelAll() {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, svc := range c.cranes {
		svc.CancelMotion()
	}
}

func (c *Coordinator) SnapshotAll() map[model.CraneID]model.CraneStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[model.CraneID]model.CraneStatus, len(c.cranes))
	for id, svc := range c.cranes {
		out[id] = svc.Status()
	}
	return out
}

func (c *Coordinator) SetEStopAll(active bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, svc := range c.cranes {
		svc.SetEStop(active)
	}
}

func (c *Coordinator) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cranes)
}

func (c *Coordinator) Planner() *path.Planner  { return c.planner }
func (c *Coordinator) Clock() *clock.DualClock { return c.clk }
