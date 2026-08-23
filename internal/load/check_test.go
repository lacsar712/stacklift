package load_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lacsar712/stacklift/internal/load"
	"github.com/lacsar712/stacklift/internal/model"
)

func TestMomentWrap(t *testing.T) {
	if !errors.Is(load.Wrap("TC-01", load.ErrMomentExceeded), load.ErrMomentExceeded) {
		t.Fatal("moment is")
	}
}

func TestValidateCancel(t *testing.T) {
	sensor := load.NewSensor()
	sensor.Put(model.LoadSample{RigID: "TC-01", MomentPct: 50, At: time.Now()})
	checker := load.NewChecker(sensor, model.DefaultLimits())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if checker.ValidateSequence(ctx, "TC-01") == nil {
		t.Fatal("cancel")
	}
}
