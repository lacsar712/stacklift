package path

import (
	"testing"

	"github.com/lacsar712/stacklift/internal/config"
	"github.com/lacsar712/stacklift/internal/model"
)

func TestCase(t *testing.T) {
	layout := config.NewLayout(config.DefaultWarehouse())
	p := NewPlanner(layout, model.DefaultLimits())
	crane := model.CraneID("CR-01")
	plan := model.MotionPlan{Steps: []model.MotionStep{{Axis: model.AxisTravel, ToMM: 1000, Order: 1}}}
	p.BindPending(crane, plan)
	exported := p.ExportPendingSteps(crane)
	exported[0].ToMM = 9999
	if plan.Steps[0].ToMM == 9999 {
		t.Fatal("exported slice aliases planner pending steps")
	}
}
