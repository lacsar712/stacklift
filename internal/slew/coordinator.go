package slew

import (
	"context"
	"fmt"
	"sync"

	"github.com/lacsar712/stacklift/internal/boom"
	"github.com/lacsar712/stacklift/internal/interlock"
	"github.com/lacsar712/stacklift/internal/load"
)

type Coordinator struct {
	mu        sync.Mutex
	plant     *Plant
	boom      *boom.Driver
	interlock *interlock.Guard
	checker   *load.Checker
	rateDeg   float64
	log       []CoordEvent
}

type CoordEvent struct {
	RigID, Kind, Detail string
}

func NewCoordinator(plant *Plant, boomDriver *boom.Driver, guard *interlock.Guard, checker *load.Checker, rateDeg float64) *Coordinator {
	return &Coordinator{plant: plant, boom: boomDriver, interlock: guard, checker: checker, rateDeg: rateDeg, log: []CoordEvent{}}
}

type RunRequest struct {
	RigID, Reason         string
	TargetAzDeg, TargetRadiusM float64
	Luff                  bool
}

func (c *Coordinator) Run(ctx context.Context, req RunRequest) error {
	if err := c.interlock.Lock(req.RigID, interlock.MotionSlew); err != nil {
		return err
	}
	defer c.interlock.Unlock(req.RigID, interlock.MotionSlew)
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.checker.ValidateSequence(ctx, req.RigID); err != nil {
		return err
	}
	if reasons := c.interlock.Eval(req.RigID, interlock.MotionSlew); len(reasons) > 0 {
		return fmt.Errorf("slew interlock: %v", reasons)
	}
	from := c.plant.emitter.AngleOf(req.RigID)
	plan := NewPlan(req.RigID, from, req.TargetAzDeg, c.rateDeg, req.Reason)
	if err := c.plant.Run(ctx, plan); err != nil {
		return err
	}
	if req.Luff && req.TargetRadiusM > 0 {
		if err := c.interlock.Lock(req.RigID, interlock.MotionLuff); err != nil {
			return err
		}
		defer c.interlock.Unlock(req.RigID, interlock.MotionLuff)
		_, err := c.boom.Move(ctx, boom.MoveRequest{RigID: req.RigID, TargetRadiusM: req.TargetRadiusM, RateDegPerS: c.rateDeg, Reason: req.Reason})
		if err != nil {
			return err
		}
	}
	return nil
}
