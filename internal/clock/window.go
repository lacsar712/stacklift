package clock

import "time"

type DwellWindow struct {
	StartTick     int64
	RequiredTicks int64
	Closed        bool
}

func NewDwellWindow(startTick, requiredTicks int64) DwellWindow {
	return DwellWindow{StartTick: startTick, RequiredTicks: requiredTicks}
}

func (w DwellWindow) IsOpen(clk *ProcessClock) bool {
	if w.Closed || clk == nil {
		return false
	}
	return !clk.WindowClosed(w.StartTick, w.RequiredTicks)
}

func (w DwellWindow) Elapsed(clk *ProcessClock) int64 {
	if clk == nil {
		return 0
	}
	return clk.ElapsedTicks(w.StartTick)
}

func (w DwellWindow) Remaining(clk *ProcessClock) int64 {
	if w.RequiredTicks <= 0 {
		return 0
	}
	elapsed := w.Elapsed(clk)
	if elapsed >= w.RequiredTicks {
		return 0
	}
	return w.RequiredTicks - elapsed
}

func (w *DwellWindow) Close() { w.Closed = true }

type WallClock struct{}

func NewWall() *WallClock { return &WallClock{} }

func (w *WallClock) Now() time.Time                       { return time.Now() }
func (w *WallClock) Since(t time.Time) time.Duration      { return time.Since(t) }
func (w *WallClock) Until(deadline time.Time) time.Duration { return time.Until(deadline) }
func (w *WallClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

type DualClock struct {
	Process *ProcessClock
	Wall    *WallClock
}

func NewDual(stepMS int64) *DualClock {
	return &DualClock{Process: NewProcess(stepMS), Wall: NewWall()}
}

func (d *DualClock) DwellSatisfied(window DwellWindow) bool {
	return d.Process.WindowClosed(window.StartTick, window.RequiredTicks)
}

func (d *DualClock) ProcessTick() int64 { return d.Process.Tick() }
func (d *DualClock) WallNow() time.Time { return d.Wall.Now() }
