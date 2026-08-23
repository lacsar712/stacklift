package clock_test

import (
	"testing"
	"time"

	"github.com/lacsar712/stacklift/internal/clock"
)

func TestProcessClockAdvance(t *testing.T) {
	pc := clock.NewProcess(100)
	if pc.Advance(5) != 5 {
		t.Fatal("advance failed")
	}
}

func TestDwellWindowProcessNotWall(t *testing.T) {
	pc := clock.NewProcess(100)
	dual := &clock.DualClock{Process: pc, Wall: clock.NewWall()}
	window := clock.NewDwellWindow(pc.Tick(), 5)
	if dual.DwellSatisfied(window) {
		t.Fatal("dwell should not be satisfied yet")
	}
	pc.Advance(5)
	if !dual.DwellSatisfied(window) {
		t.Fatal("dwell should be satisfied after process ticks")
	}
	time.Sleep(2 * time.Millisecond)
}

func TestPauseResume(t *testing.T) {
	pc := clock.NewProcess(100)
	pc.Advance(3)
	pc.Pause()
	if pc.Advance(10) != 3 {
		t.Fatal("paused clock should not advance")
	}
	pc.Resume()
	if pc.Advance(2) != 5 {
		t.Fatal("expected 5 after resume")
	}
}

func TestWindowClosed(t *testing.T) {
	pc := clock.NewProcess(100)
	start := pc.Tick()
	pc.Advance(3)
	if pc.WindowClosed(start, 5) {
		t.Fatal("window should be open")
	}
	pc.Advance(2)
	if !pc.WindowClosed(start, 5) {
		t.Fatal("window should be closed")
	}
}
