package telemetry_test

import (
	"testing"

	"github.com/lacsar712/stacklift/internal/telemetry"
)

func TestBus(t *testing.T) {
	b := telemetry.NewBus(8)
	b.Emit("s", "k", "TC-01", 1, "d")
	if len(b.Snapshot()) != 1 {
		t.Fatal("event")
	}
}

func TestCounters(t *testing.T) {
	c := telemetry.NewCounters()
	c.Inc("x", 1)
	if c.Snapshot()["x"] != 1 {
		t.Fatal("counter")
	}
}
