package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lacsar712/stacklift/internal/app"
	"github.com/lacsar712/stacklift/internal/config"
	"github.com/lacsar712/stacklift/internal/load"
	"github.com/lacsar712/stacklift/internal/model"
)

func TestNew(t *testing.T) {
	svc, err := app.New(config.Default())
	if err != nil {
		t.Fatal(err)
	}
	if svc.ProcessTick() != 0 {
		t.Fatal("tick")
	}
}

func TestSlew(t *testing.T) {
	svc, _ := app.New(config.Default())
	svc.IngestLoad(model.LoadSample{RigID: "TC-01", MomentPct: 50, At: time.Now()})
	if err := svc.RequestSlew(context.Background(), "TC-01", 20, 0, "t"); err != nil {
		t.Fatal(err)
	}
}

func TestSentinels(t *testing.T) {
	svc, _ := app.New(config.Default())
	if !svc.MomentExceeded(load.Wrap("TC-01", load.ErrMomentExceeded)) {
		t.Fatal("moment")
	}
	if !svc.StaleLoad(load.Wrap("TC-01", load.ErrStaleLoad)) {
		t.Fatal("stale")
	}
}

func TestEmergency(t *testing.T) {
	svc, _ := app.New(config.Default())
	if err := svc.EmergencyStop("TC-01"); err != nil {
		t.Fatal(err)
	}
}

func TestMomentChain(t *testing.T) {
	if !errors.Is(load.Wrap("TC-01", load.ErrMomentExceeded), load.ErrMomentExceeded) {
		t.Fatal("chain")
	}
}
