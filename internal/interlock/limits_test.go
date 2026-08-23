package interlock_test

import (
	"errors"
	"testing"

	"github.com/lacsar712/stacklift/internal/clock"
	"github.com/lacsar712/stacklift/internal/interlock"
	"github.com/lacsar712/stacklift/internal/load"
	"github.com/lacsar712/stacklift/internal/model"
)

func TestLockUnlock(t *testing.T) {
	limits := interlock.Limits{WindHoldTicks: 5}
	clk := clock.New(100)
	g := interlock.NewGuard(limits, interlock.NewWindWindow(limits, clk))
	if err := g.Lock("TC-01", interlock.MotionSlew); err != nil {
		t.Fatal(err)
	}
	g.Unlock("TC-01", interlock.MotionSlew)
}

func TestStaleLoad(t *testing.T) {
	limits := interlock.Limits{}
	clk := clock.New(100)
	g := interlock.NewGuard(limits, interlock.NewWindWindow(limits, clk))
	g.SetLoadFault("TC-01", load.Wrap("TC-01", load.ErrStaleLoad))
	if !errors.Is(g.StaleLoadCheck("TC-01"), load.ErrStaleLoad) {
		t.Fatal("stale")
	}
}

func TestWindWindow(t *testing.T) {
	limits := interlock.Limits{WindHoldTicks: 3}
	clk := clock.New(100)
	w := interlock.NewWindWindow(limits, clk)
	w.Open("TC-01")
	clk.Advance(5)
	if !w.Closed("TC-01") {
		t.Fatal("closed")
	}
}

func TestEmergencyInhibit(t *testing.T) {
	limits := interlock.Limits{}
	clk := clock.New(100)
	g := interlock.NewGuard(limits, interlock.NewWindWindow(limits, clk))
	g.SetMode("TC-01", model.DutyEmergency)
	if len(g.Eval("TC-01", interlock.MotionSlew)) == 0 {
		t.Fatal("inhibit")
	}
}
