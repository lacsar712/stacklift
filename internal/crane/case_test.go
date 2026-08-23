package crane_test

import (
	"context"
	"testing"

	"github.com/lacsar712/stacklift/internal/clock"
	"github.com/lacsar712/stacklift/internal/config"
	"github.com/lacsar712/stacklift/internal/crane"
	"github.com/lacsar712/stacklift/internal/model"
	"github.com/lacsar712/stacklift/internal/path"
)

func setup(t *testing.T) (*crane.Coordinator, *crane.Service) {
	t.Helper()
	clk := clock.NewDual(100)
	layout := config.NewLayout(config.DefaultWarehouse())
	planner := path.NewPlanner(layout, model.DefaultLimits())
	coord := crane.NewCoordinator(planner, clk, model.DefaultLimits())
	svc := coord.Register("CR-01")
	return coord, svc
}

func TestCase(t *testing.T) {
	_, svc := setup(t)
	if err := svc.MoveAxis(context.Background(), model.AxisTravel, 1000); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	plan := model.MotionPlan{Steps: []model.MotionStep{
		{Axis: model.AxisHoist, ToMM: 2500},
	}}
	if err := crane.ExecutePlan(ctx, svc, plan); err == nil && svc.Pose().HoistMM >= 2500 {
		t.Fatal("hoist completed after task cancel")
	}
	if svc.Pose().HoistMM >= 2500 {
		t.Fatalf("hoist mm=%d should not reach target after cancel", svc.Pose().HoistMM)
	}
}
