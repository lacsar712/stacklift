package fsm_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lacsar712/stacklift/internal/clock"
	"github.com/lacsar712/stacklift/internal/config"
	"github.com/lacsar712/stacklift/internal/crane"
	"github.com/lacsar712/stacklift/internal/fsm"
	"github.com/lacsar712/stacklift/internal/model"
	"github.com/lacsar712/stacklift/internal/path"
)

func setup(t *testing.T) (*fsm.MotionMachine, *crane.Coordinator) {
	t.Helper()
	clk := clock.NewDual(100)
	layout := config.NewLayout(config.DefaultWarehouse())
	planner := path.NewPlanner(layout, model.DefaultLimits())
	coord := crane.NewCoordinator(planner, clk, model.DefaultLimits())
	coord.Register("CR-01")
	machine := fsm.NewMotionMachine()
	fsm.RegisterDefaultEffects(machine, coord, clk, model.DefaultLimits())
	return machine, coord
}

func TestTransition(t *testing.T) {
	m, _ := setup(t)
	ctx := context.Background()
	res, err := m.Fire(ctx, "CR-01", model.EventStartTravel)
	if err != nil || !res.Accepted || res.Phase != model.PhaseTraveling {
		t.Fatalf("result=%+v err=%v", res, err)
	}
	res, err = m.Fire(ctx, "CR-01", model.EventTravelDone)
	if err != nil || res.Phase != model.PhaseIdle {
		t.Fatalf("done: %+v err=%v", res, err)
	}
}

func TestIllegalTransition(t *testing.T) {
	m, _ := setup(t)
	_, err := m.Fire(context.Background(), "CR-01", model.EventHoistDone)
	if err == nil || !errors.Is(err, model.ErrInvalidPhase) {
		t.Fatalf("expected ErrInvalidPhase got %v", err)
	}
}

func TestSideEffects(t *testing.T) {
	m, coord := setup(t)
	_, _ = m.Fire(context.Background(), "CR-01", model.EventStartTravel)
	svc, _ := coord.Get("CR-01")
	if svc.Status().Phase != model.PhaseTraveling {
		t.Fatal("side effect should set traveling phase")
	}
}

func TestEStopSideEffect(t *testing.T) {
	m, coord := setup(t)
	_, _ = m.Fire(context.Background(), "CR-01", model.EventEStop)
	svc, _ := coord.Get("CR-01")
	if !svc.Status().EStopActive {
		t.Fatal("estop side effect failed")
	}
}

func TestCanTransition(t *testing.T) {
	if !fsm.CanTransition(model.PhaseIdle, model.EventStartHoist) {
		t.Fatal("should allow hoist from idle")
	}
}

func TestDwellTick(t *testing.T) {
	m, _ := setup(t)
	m.SetDwellTick("CR-01", 42)
	tick, ok := m.DwellTick("CR-01")
	if !ok || tick != 42 {
		t.Fatalf("tick=%d ok=%v", tick, ok)
	}
}

func TestAllowedEvents(t *testing.T) {
	if len(fsm.AllowedEvents(model.PhaseIdle)) == 0 {
		t.Fatal("idle should have allowed events")
	}
}
