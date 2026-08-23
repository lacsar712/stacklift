package slew_test

import (
	"context"
	"testing"
	"time"

	"github.com/lacsar712/stacklift/internal/boom"
	"github.com/lacsar712/stacklift/internal/clock"
	"github.com/lacsar712/stacklift/internal/interlock"
	"github.com/lacsar712/stacklift/internal/load"
	"github.com/lacsar712/stacklift/internal/model"
	"github.com/lacsar712/stacklift/internal/slew"
)

func TestCoordinatorRun(t *testing.T) {
	e := slew.NewEmitter()
	e.SetAngle("TC-01", 0)
	plant := slew.NewPlant(e, 5)
	g := boom.NewGeometry(5, 60, 15, 75)
	driver := boom.NewDriver(g, 40, 5)
	driver.Ensure("TC-01")
	limits := interlock.Limits{MaxMomentPct: 100, WindHoldTicks: 10, SlewBoomMutex: true}
	clk := clock.New(100)
	guard := interlock.NewGuard(limits, interlock.NewWindWindow(limits, clk))
	sensor := load.NewSensor()
	sensor.Put(model.LoadSample{RigID: "TC-01", MomentPct: 40, At: time.Now()})
	checker := load.NewChecker(sensor, model.DefaultLimits())
	coord := slew.NewCoordinator(plant, driver, guard, checker, 5)
	if err := coord.Run(context.Background(), slew.RunRequest{RigID: "TC-01", TargetAzDeg: 10, Reason: "t"}); err != nil {
		t.Fatal(err)
	}
}
