package interlock

import (
	"context"
	"errors"
	"fmt"

	"github.com/lacsar712/stacklift/internal/clock"
	"github.com/lacsar712/stacklift/internal/model"
)

type Guard struct {
	limits model.LimitSet
	clk     *clock.DualClock
	dwells  map[model.CraneID]clock.DwellWindow
	hoistLease map[model.CraneID]bool
}

func NewGuard(limits model.LimitSet, clk *clock.DualClock) *Guard {
	return &Guard{limits: limits, clk: clk, dwells: make(map[model.CraneID]clock.DwellWindow), hoistLease: make(map[model.CraneID]bool)}
}

func (g *Guard) AcquireHoistLease(crane model.CraneID) error {
	if g.hoistLease[crane] {
		return model.ErrHoistLocked
	}
	g.hoistLease[crane] = true
	return nil
}

func (g *Guard) ReleaseHoistLease(crane model.CraneID) {
	delete(g.hoistLease, crane)
}

func (g *Guard) CheckTravel(status model.CraneStatus) error {
	if status.EStopActive {
		return model.ErrEStopActive
	}
	if status.Pose.ForkPos != model.ForkRetracted {
		return model.NewInterlockError("travel_fork", status.CraneID, "fork must be retracted before travel")
	}
	if status.Phase == model.PhaseHoisting || status.Phase == model.PhaseForking {
		return model.NewInterlockError("travel_axis", status.CraneID, "hoist or fork in progress")
	}
	return nil
}

func (g *Guard) CheckHoist(status model.CraneStatus) error {
	if status.EStopActive {
		return model.ErrEStopActive
	}
	if status.Pose.ForkPos == model.ForkExtended {
		return model.NewInterlockError("hoist_fork", status.CraneID, "fork extended blocks hoist")
	}
	if status.Phase == model.PhaseTraveling {
		return model.NewInterlockError("hoist_travel", status.CraneID, "travel in progress")
	}
	if status.Pose.HoistMM < g.limits.MinHoistClearMM && status.Pose.ForkPos != model.ForkRetracted {
		return model.NewInterlockError("hoist_clearance", status.CraneID, "insufficient hoist clearance")
	}
	return nil
}

func (g *Guard) CheckFork(status model.CraneStatus) error {
	if status.EStopActive {
		return model.ErrEStopActive
	}
	if status.Phase == model.PhaseTraveling {
		return model.NewInterlockError("fork_travel", status.CraneID, "travel in progress")
	}
	if status.Phase == model.PhaseHoisting {
		return model.NewInterlockError("fork_hoist", status.CraneID, "hoist in progress")
	}
	return nil
}

func (g *Guard) CheckAxis(status model.CraneStatus, axis model.MotionAxis) error {
	switch axis {
	case model.AxisTravel:
		return g.CheckTravel(status)
	case model.AxisHoist:
		return g.CheckHoist(status)
	case model.AxisFork:
		return g.CheckFork(status)
	default:
		return fmt.Errorf("unknown axis: %w", model.ErrAxisConflict)
	}
}

func (g *Guard) StartDwell(crane model.CraneID) {
	g.dwells[crane] = clock.NewDwellWindow(g.clk.ProcessTick(), g.limits.DwellWindowTicks)
}

// DwellComplete judges effective dwell against the motion process clock, which
// only advances while the motion loop is running. This keeps confirmation
// consistent with the process-clock accumulation used elsewhere and prevents
// wall-clock drift (e.g. communication/latency compensation) from closing the
// window early before the minimum dwell has been reached.
func (g *Guard) DwellComplete(crane model.CraneID) bool {
	w, ok := g.dwells[crane]
	if !ok {
		return true
	}
	return g.clk.DwellSatisfied(w)
}

func (g *Guard) DwellRemaining(crane model.CraneID) int64 {
	w, ok := g.dwells[crane]
	if !ok {
		return 0
	}
	return w.Remaining(g.clk.Process)
}

func (g *Guard) ClearDwell(crane model.CraneID) { delete(g.dwells, crane) }

func (g *Guard) ValidateMotion(ctx context.Context, status model.CraneStatus, axis model.MotionAxis) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("interlock: %w", model.ErrContextDone)
	default:
	}
	if err := g.CheckAxis(status, axis); err != nil {
		return err
	}
	if status.Interlocked {
		return model.NewInterlockError("global", status.CraneID, "crane globally interlocked")
	}
	return nil
}

func (g *Guard) IsBlocked(err error) bool { return errors.Is(err, model.ErrInterlockBlocked) }
