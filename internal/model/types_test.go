package model_test

import (
	"errors"
	"testing"

	"github.com/lacsar712/stacklift/internal/model"
)

func TestLocationValid(t *testing.T) {
	loc := model.Location{Aisle: "01", Bay: 1, Level: 1, Depth: 0}
	if !loc.Valid() {
		t.Fatal("expected valid location")
	}
	empty := model.Location{}
	if empty.Valid() {
		t.Fatal("expected invalid location")
	}
}

func TestCloneProtection(t *testing.T) {
	snap := model.NewWarehouseSnapshot(1, 10)
	snap.Cranes = map[model.CraneID]model.CraneStatus{"CR-01": {CraneID: "CR-01", Phase: model.PhaseIdle}}
	clone := snap.Clone()
	clone.Cranes["CR-01"] = model.CraneStatus{CraneID: "CR-01", Phase: model.PhaseFault}
	orig, _ := snap.Crane("CR-01")
	if orig.Phase == model.PhaseFault {
		t.Fatal("clone mutation affected original")
	}
}

func TestNextPhase(t *testing.T) {
	next, ok := model.NextPhase(model.PhaseIdle, model.EventStartTravel)
	if !ok || next != model.PhaseTraveling {
		t.Fatalf("got %s ok=%v", next, ok)
	}
}

func TestErrorChain(t *testing.T) {
	err := model.NewMotionError(model.AxisHoist, "CR-01", model.ErrHoistLocked)
	if !errors.Is(err, model.ErrHoistLocked) {
		t.Fatal("expected ErrHoistLocked in chain")
	}
	il := model.NewInterlockError("test", "CR-01", "blocked")
	if !errors.Is(il, model.ErrInterlockBlocked) {
		t.Fatal("expected ErrInterlockBlocked in chain")
	}
}

func TestDefaultLimits(t *testing.T) {
	if err := model.DefaultLimits().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestPositionConversion(t *testing.T) {
	loc := model.Location{Aisle: "01", Bay: 2, Level: 3, Depth: 1}
	pos := model.LocationToPosition(loc, 2800, 1800, 1200)
	if pos.TravelMM != 5600 {
		t.Fatalf("travel=%d", pos.TravelMM)
	}
}

func TestMotionPlanClone(t *testing.T) {
	plan := model.MotionPlan{Steps: []model.MotionStep{{Axis: model.AxisTravel, FromMM: 0, ToMM: 1000}}}
	clone := plan.Clone()
	clone.Steps[0].ToMM = 9999
	if plan.Steps[0].ToMM == 9999 {
		t.Fatal("clone mutated original steps")
	}
}
