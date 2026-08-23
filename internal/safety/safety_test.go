package safety_test

import (
	"testing"

	"github.com/lacsar712/stacklift/internal/clock"
	"github.com/lacsar712/stacklift/internal/config"
	"github.com/lacsar712/stacklift/internal/crane"
	"github.com/lacsar712/stacklift/internal/model"
	"github.com/lacsar712/stacklift/internal/path"
	"github.com/lacsar712/stacklift/internal/safety"
)

func setup() (*safety.EStopController, *crane.Coordinator) {
	clk := clock.NewDual(100)
	layout := config.NewLayout(config.DefaultWarehouse())
	planner := path.NewPlanner(layout, model.DefaultLimits())
	coord := crane.NewCoordinator(planner, clk, model.DefaultLimits())
	coord.Register("CR-01")
	monitor := safety.NewMonitor(model.DefaultLimits())
	return safety.NewEStopController(coord, monitor), coord
}

func TestMonitorAlarms(t *testing.T) {
	m := safety.NewMonitor(model.DefaultLimits())
	m.Raise("CR-01", "TEST", model.AlarmWarn, "test alarm")
	if m.AlarmCount() != 1 {
		t.Fatal("expected 1 alarm")
	}
}

func TestCheckPoseLimits(t *testing.T) {
	m := safety.NewMonitor(model.DefaultLimits())
	pose := model.CranePose{TravelMM: model.DefaultLimits().MaxTravelMM + 100}
	if len(m.CheckPose("CR-01", pose)) == 0 {
		t.Fatal("expected limit alarm")
	}
}

func TestEStopTriggerReset(t *testing.T) {
	estop, coord := setup()
	estop.Trigger()
	if !estop.Active() {
		t.Fatal("estop should be active")
	}
	svc, _ := coord.Get("CR-01")
	if !svc.Status().EStopActive {
		t.Fatal("crane should have estop")
	}
	if err := estop.Reset(); err != nil {
		t.Fatal(err)
	}
}

func TestEStopTriggeredCranes(t *testing.T) {
	estop, _ := setup()
	estop.Trigger()
	if len(estop.TriggeredCranes()) != 1 {
		t.Fatal("expected 1 triggered crane")
	}
}

func TestMonitorClear(t *testing.T) {
	m := safety.NewMonitor(model.DefaultLimits())
	m.Raise("CR-01", "A", model.AlarmInfo, "info")
	m.Clear()
	if m.AlarmCount() != 0 {
		t.Fatal("expected cleared alarms")
	}
}
