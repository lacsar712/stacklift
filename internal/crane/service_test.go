package crane_test

import (
	"context"
	"testing"
	"time"

	"github.com/lacsar712/stacklift/internal/boom"
	"github.com/lacsar712/stacklift/internal/clock"
	"github.com/lacsar712/stacklift/internal/crane"
	"github.com/lacsar712/stacklift/internal/fsm"
	"github.com/lacsar712/stacklift/internal/interlock"
	"github.com/lacsar712/stacklift/internal/load"
	"github.com/lacsar712/stacklift/internal/model"
	"github.com/lacsar712/stacklift/internal/slew"
	"github.com/lacsar712/stacklift/internal/store"
)

func TestRequestSlew(t *testing.T) {
	e := slew.NewEmitter()
	plant := slew.NewPlant(e, 5)
	g := boom.NewGeometry(5, 60, 15, 75)
	driver := boom.NewDriver(g, 40, 5)
	limits := interlock.Limits{MaxMomentPct: 100, WindHoldTicks: 10, SlewBoomMutex: true}
	clk := clock.New(100)
	guard := interlock.NewGuard(limits, interlock.NewWindWindow(limits, clk))
	sensor := load.NewSensor()
	sensor.Put(model.LoadSample{RigID: "TC-01", MomentPct: 40, At: time.Now()})
	coord := slew.NewCoordinator(plant, driver, guard, load.NewChecker(sensor, model.DefaultLimits()), 5)
	machine := fsm.NewDutyMachine(e)
	st := store.NewRigStore()
	group := crane.NewGroup()
	rig := crane.NewBuilder(40, 12000, 5, model.DefaultLimits(), 10).Build("TC-01", e, machine, st)
	group.Register(rig)
	svc := crane.NewService(group, coord, sensor, machine, st)
	svc.Bootstrap()
	if err := svc.RequestSlew(context.Background(), "TC-01", 15, 0, "t"); err != nil {
		t.Fatal(err)
	}
}
