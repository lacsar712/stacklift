package hook_test

import (
	"testing"

	"github.com/lacsar712/stacklift/internal/hook"
)

func TestSensor(t *testing.T) {
	s := hook.NewSensor(10000)
	p := s.Ingest("TC-01", 5000, 30, 20, 1)
	if p.MomentPct <= 0 {
		t.Fatal("moment")
	}
}

func TestEstimator(t *testing.T) {
	e := hook.NewEstimator(40, 60, 10000)
	if e.MomentPercent(5000, 30) <= 0 {
		t.Fatal("pct")
	}
}
