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

func TestCase(t *testing.T) {
	clk := clock.NewDual(100)
	layout := config.NewLayout(config.DefaultWarehouse())
	planner := path.NewPlanner(layout, model.DefaultLimits())
	coord := crane.NewCoordinator(planner, clk, model.DefaultLimits())
	svc := coord.Register("CR-01")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	plan := model.MotionPlan{Steps: []model.MotionStep{
		{Axis: model.AxisTravel, ToMM: 1000},
		{Axis: model.AxisHoist, ToMM: 2000},
	}}
	stepsBefore := 0
	_ = crane.ExecutePlan(ctx, svc, plan)
	if svc.Pose().TravelMM > 0 && svc.Pose().HoistMM > 0 {
		t.Fatal("motion continued after cancelled ctx")
	}
	_ = stepsBefore
}
