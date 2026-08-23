package interlock

import (
	"testing"

	"github.com/lacsar712/stacklift/internal/clock"
	"github.com/lacsar712/stacklift/internal/model"
)

func TestCase(t *testing.T) {
	clk := clock.NewDual(100)
	g := NewGuard(model.DefaultLimits(), clk)
	crane := model.CraneID("CR-01")
	g.StartDwell(crane)
	if g.DwellComplete(crane) {
		t.Fatal("dwell should not complete without process ticks")
	}
	clk.Process.Advance(100)
	if !g.DwellComplete(crane) {
		t.Fatal("dwell should complete after enough process ticks")
	}
}
