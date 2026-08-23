package clock_test

import (
	"testing"

	"github.com/lacsar712/stacklift/internal/clock"
)

func TestAdvance(t *testing.T) {
	c := clock.New(100)
	c.Advance(3)
	if c.Tick() != 3 {
		t.Fatal(c.Tick())
	}
}

func TestWindowClosed(t *testing.T) {
	c := clock.New(100)
	start := c.Tick()
	c.Advance(10)
	if !c.WindowClosed(start, 5) {
		t.Fatal("should close")
	}
}
