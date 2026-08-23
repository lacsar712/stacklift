package interlock_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lacsar712/stacklift/internal/clock"
	"github.com/lacsar712/stacklift/internal/interlock"
	"github.com/lacsar712/stacklift/internal/model"
)

func setup() (*interlock.Guard, *clock.DualClock) {
	clk := clock.NewDual(100)
	return interlock.NewGuard(model.DefaultLimits(), clk), clk
}

func TestTravelBlockedByFork(t *testing.T) {
	guard, _ := setup()
	status := model.CraneStatus{CraneID: "CR-01", Pose: model.CranePose{ForkPos: model.ForkExtended}}
	err := guard.CheckTravel(status)
	if err == nil || !errors.Is(err, model.ErrInterlockBlocked) {
		t.Fatal("expected interlock error")
	}
}

func TestHoistBlockedByTravel(t *testing.T) {
	guard, _ := setup()
	status := model.CraneStatus{CraneID: "CR-01", Phase: model.PhaseTraveling, Pose: model.CranePose{ForkPos: model.ForkRetracted}}
	if guard.CheckHoist(status) == nil {
		t.Fatal("expected hoist blocked")
	}
}

func TestForkBlockedByHoist(t *testing.T) {
	guard, _ := setup()
	status := model.CraneStatus{CraneID: "CR-01", Phase: model.PhaseHoisting}
	if guard.CheckFork(status) == nil {
		t.Fatal("expected fork blocked")
	}
}

func TestDwellWindow(t *testing.T) {
	guard, clk := setup()
	guard.StartDwell("CR-01")
	clk.Process.Advance(model.DefaultLimits().DwellWindowTicks)
	if !guard.DwellComplete("CR-01") {
		t.Fatal("dwell should be complete after ticks")
	}
}

func TestValidateMotionContext(t *testing.T) {
	guard, _ := setup()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := guard.ValidateMotion(ctx, model.EmptyCraneStatus("CR-01"), model.AxisTravel)
	if !errors.Is(err, model.ErrContextDone) {
		t.Fatalf("expected context done got %v", err)
	}
}

func TestRuleSet(t *testing.T) {
	guard, _ := setup()
	rs := interlock.NewRuleSet(guard)
	if rs.Count() < 3 {
		t.Fatal("expected at least 3 rules")
	}
}

func TestIsBlocked(t *testing.T) {
	guard, _ := setup()
	status := model.CraneStatus{CraneID: "CR-01", Pose: model.CranePose{ForkPos: model.ForkExtended}}
	if !guard.IsBlocked(guard.CheckTravel(status)) {
		t.Fatal("should be blocked")
	}
}
