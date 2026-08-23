package wind_test

import (
	"errors"
	"testing"

	"github.com/lacsar712/stacklift/internal/model"
	"github.com/lacsar712/stacklift/internal/wind"
)

func TestGustWrap(t *testing.T) {
	err := wind.Wrap("TC-01", wind.ErrWindGust)
	if !errors.Is(err, wind.ErrWindGust) {
		t.Fatal("is")
	}
}

func TestSustained(t *testing.T) {
	a := wind.NewAnemometer(13.8, 1.35, 5)
	err := a.Ingest(model.WindSample{RigID: "TC-01", SpeedMS: 20, ProcessTick: 1})
	if !errors.Is(err, wind.ErrSustainedHigh) {
		t.Fatal(err)
	}
}
