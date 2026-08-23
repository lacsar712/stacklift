package fsm_test

import (
	"context"
	"testing"

	"github.com/lacsar712/stacklift/internal/clock"
	"github.com/lacsar712/stacklift/internal/config"
	"github.com/lacsar712/stacklift/internal/crane"
	"github.com/lacsar712/stacklift/internal/fsm"
	"github.com/lacsar712/stacklift/internal/model"
	"github.com/lacsar712/stacklift/internal/path"
)

func TestCase(t *testing.T) {
	clk := clock.NewDual(100)
	layout := config.NewLayout(config.DefaultWarehouse())
	planner := path.NewPlanner(layout, model.DefaultLimits())
	coord := crane.NewCoordinator(planner, clk, model.DefaultLimits())
	svc := coord.Register("CR-01")
	machine := fsm.NewMotionMachine()
	fsm.RegisterDefaultEffects(machine, coord, clk, model.DefaultLimits())
	machine.Ensure("CR-01")
	before := svc.Status().Phase
	_, err := machine.Fire(context.Background(), "CR-01", model.EventLoadPallet)
	if err == nil {
		t.Fatal("expected illegal transition")
	}
	if svc.Status().Phase != before {
		t.Fatalf("phase changed on illegal transition: %s -> %s", before, svc.Status().Phase)
	}
}
