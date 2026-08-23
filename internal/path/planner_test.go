package path_test

import (
	"testing"

	"github.com/lacsar712/stacklift/internal/config"
	"github.com/lacsar712/stacklift/internal/model"
	"github.com/lacsar712/stacklift/internal/path"
)

func TestPlanner(t *testing.T) {
	layout := config.NewLayout(config.DefaultWarehouse())
	planner := path.NewPlanner(layout, model.DefaultLimits())
	from := model.Location{Aisle: "01", Bay: 1, Level: 1, Depth: 0}
	to := model.Location{Aisle: "01", Bay: 5, Level: 3, Depth: 1}
	plan, err := planner.Plan("T1", "CR-01", from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Steps) < 2 {
		t.Fatalf("expected multiple steps got %d", len(plan.Steps))
	}
}

func TestRoute(t *testing.T) {
	plan := model.MotionPlan{Steps: []model.MotionStep{
		{Axis: model.AxisTravel, Order: 1},
		{Axis: model.AxisHoist, Order: 2},
	}}
	route := path.NewRoute(plan)
	if _, ok := route.Current(); !ok {
		t.Fatal("expected step")
	}
	route.Advance()
	route.Advance()
	if !route.Done {
		t.Fatal("route should be done")
	}
}

func TestCrossAisleRejected(t *testing.T) {
	layout := config.NewLayout(config.DefaultWarehouse())
	planner := path.NewPlanner(layout, model.DefaultLimits())
	from := model.Location{Aisle: "01", Bay: 1, Level: 1, Depth: 0}
	to := model.Location{Aisle: "02", Bay: 1, Level: 1, Depth: 0}
	if _, err := planner.Plan("T1", "CR-01", from, to); err == nil {
		t.Fatal("expected cross-aisle error")
	}
}

func TestRouteProgress(t *testing.T) {
	plan := model.MotionPlan{Steps: []model.MotionStep{{}, {}}}
	route := path.NewRoute(plan)
	route.Advance()
	if route.Progress() != 0.5 {
		t.Fatalf("expected 0.5 got %f", route.Progress())
	}
}

func TestEstimateDistance(t *testing.T) {
	layout := config.NewLayout(config.DefaultWarehouse())
	planner := path.NewPlanner(layout, model.DefaultLimits())
	from := model.Location{Aisle: "01", Bay: 1, Level: 1, Depth: 0}
	to := model.Location{Aisle: "01", Bay: 2, Level: 1, Depth: 0}
	if planner.EstimateDistance(from, to) <= 0 {
		t.Fatal("expected positive distance")
	}
}
