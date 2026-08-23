package clock

import "time"

type Timer struct {
	wall *WallClock
}

func NewTimer() *Timer { return &Timer{wall: NewWall()} }

func (t *Timer) DeadlineAfter(d time.Duration) time.Time { return t.wall.Now().Add(d) }
func (t *Timer) Expired(deadline time.Time) bool       { return !t.wall.Now().Before(deadline) }
func (t *Timer) Remaining(deadline time.Time) time.Duration { return t.wall.Until(deadline) }
func (t *Timer) Sleep(d time.Duration)                     { time.Sleep(d) }
