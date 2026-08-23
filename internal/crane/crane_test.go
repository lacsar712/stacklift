package crane_test

import (
	"context"
	"errors"
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

func TestMoveAxis(t *testing.T) {
	_, svc := setup(t)
	if err := svc.MoveAxis(context.Background(), model.AxisTravel, 5000); err != nil {
		t.Fatal(err)
	}
	if svc.Pose().TravelMM != 5000 {
		t.Fatal("travel not updated")
	}
}

func TestContextCancel(t *testing.T) {
	_, svc := setup(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := svc.MoveAxis(ctx, model.AxisHoist, 1000)
	if err == nil || !errors.Is(err, model.ErrContextDone) {
		t.Fatalf("expected context error got %v", err)
	}
}

func TestHoistLockDefer(t *testing.T) {
	_, svc := setup(t)
	if svc.HoistLocked() {
		t.Fatal("hoist should start unlocked")
	}
	if err := svc.MoveAxis(context.Background(), model.AxisHoist, 2000); err != nil {
		t.Fatal(err)
	}
	if svc.HoistLocked() {
		t.Fatal("hoist should be unlocked after motion")
	}
}

func TestEStop(t *testing.T) {
	_, svc := setup(t)
	svc.SetEStop(true)
	if !errors.Is(svc.MoveAxis(context.Background(), model.AxisTravel, 1000), model.ErrEStopActive) {
		t.Fatal("expected estop error")
	}
}

func TestLoadUnload(t *testing.T) {
	_, svc := setup(t)
	if err := svc.LoadPallet("PLT-1"); err != nil {
		t.Fatal(err)
	}
	id, err := svc.UnloadPallet()
	if err != nil || id != "PLT-1" {
		t.Fatalf("unload: id=%s err=%v", id, err)
	}
}

func TestCoordinatorMove(t *testing.T) {
	coord, _ := setup(t)
	from := model.Location{Aisle: "01", Bay: 1, Level: 1, Depth: 0}
	to := model.Location{Aisle: "01", Bay: 3, Level: 2, Depth: 0}
	if err := coord.MoveCrane(context.Background(), "CR-01", from, to); err != nil {
		t.Fatal(err)
	}
}

func TestExecutePlan(t *testing.T) {
	_, svc := setup(t)
	plan := model.MotionPlan{Steps: []model.MotionStep{
		{Axis: model.AxisTravel, ToMM: 2800},
		{Axis: model.AxisHoist, ToMM: 1800},
	}}
	if err := crane.ExecutePlan(context.Background(), svc, plan); err != nil {
		t.Fatal(err)
	}
}
