// Package interlock enforces slew/boom mutual inhibition and wind hold.
package interlock

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lacsar712/stacklift/internal/clock"
	"github.com/lacsar712/stacklift/internal/config"
	"github.com/lacsar712/stacklift/internal/load"
	"github.com/lacsar712/stacklift/internal/model"
	"github.com/lacsar712/stacklift/internal/wind"
)

var (
	ErrLocked    = errors.New("interlock: locked")
	ErrInhibited = errors.New("interlock: inhibited")
)

type MotionKind string

const (
	MotionSlew  MotionKind = "slew"
	MotionLuff  MotionKind = "luff"
	MotionHoist MotionKind = "hoist"
)

type Limits struct {
	MaxMomentPct, WarnMomentPct, MaxWindMS, WindHoldTicks float64
	SlewBoomMutex                                           bool
	MomentDeratePct                                         float64
}

func LimitsFromConfig(cfg config.Config) Limits {
	return Limits{
		MaxMomentPct: cfg.Limits.MaxMomentPct, WarnMomentPct: cfg.Limits.WarnMomentPct,
		MaxWindMS: cfg.Limits.MaxWindMS, WindHoldTicks: float64(cfg.WindHoldWindow),
		SlewBoomMutex: cfg.Interlock.SlewBoomMutex, MomentDeratePct: cfg.Interlock.MomentDeratePct,
	}
}

type WindWindow struct {
	mu        sync.Mutex
	limits    Limits
	clock     *clock.ProcessClock
	startTick map[string]int64
	active    map[string]bool
}

func NewWindWindow(limits Limits, clk *clock.ProcessClock) *WindWindow {
	return &WindWindow{limits: limits, clock: clk, startTick: make(map[string]int64), active: make(map[string]bool)}
}

func (w *WindWindow) Open(rigID string) {
	w.mu.Lock()
	w.startTick[rigID] = w.clock.Tick()
	w.active[rigID] = true
	w.mu.Unlock()
}

func (w *WindWindow) Closed(rigID string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.active[rigID] {
		return true
	}
	start, ok := w.startTick[rigID]
	if !ok {
		return true
	}
	return w.clock.WindowClosed(start, int64(w.limits.WindHoldTicks))
}

func (w *WindWindow) Active(rigID string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.active[rigID]
}

func (w *WindWindow) ElapsedTicks(rigID string) int64 {
	w.mu.Lock()
	start, ok := w.startTick[rigID]
	w.mu.Unlock()
	if !ok {
		return 0
	}
	return w.clock.ElapsedTicks(start)
}

type Guard struct {
	mu         sync.Mutex
	limits     Limits
	locks      map[string]map[MotionKind]bool
	windWindow *WindWindow
	windFault  map[string]error
	loadFault  map[string]error
	mode       map[string]model.DutyMode
}

func NewGuard(limits Limits, windWindow *WindWindow) *Guard {
	return &Guard{
		limits: limits, locks: make(map[string]map[MotionKind]bool),
		windWindow: windWindow, windFault: make(map[string]error),
		loadFault: make(map[string]error), mode: make(map[string]model.DutyMode),
	}
}

func (g *Guard) SetMode(rigID string, mode model.DutyMode) {
	g.mu.Lock()
	g.mode[rigID] = mode
	g.mu.Unlock()
}

func (g *Guard) SetWindFault(rigID string, err error) {
	g.mu.Lock()
	g.windFault[rigID] = err
	if err != nil && wind.IsBan(err) {
		g.windWindow.Open(rigID)
	}
	g.mu.Unlock()
}

func (g *Guard) SetLoadFault(rigID string, err error) {
	g.mu.Lock()
	g.loadFault[rigID] = err
	g.mu.Unlock()
}

func (g *Guard) Lock(rigID string, kind MotionKind) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.locks[rigID] == nil {
		g.locks[rigID] = make(map[MotionKind]bool)
	}
	if g.locks[rigID][kind] {
		return fmt.Errorf("%w: %s already locked for %s", ErrLocked, kind, rigID)
	}
	if g.limits.SlewBoomMutex {
		for k, v := range g.locks[rigID] {
			if v {
				return fmt.Errorf("%w: mutex %s blocks %s", ErrInhibited, k, kind)
			}
		}
	}
	g.locks[rigID][kind] = true
	return nil
}

func (g *Guard) Unlock(rigID string, kind MotionKind) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.locks[rigID] != nil {
		delete(g.locks[rigID], kind)
	}
}

func (g *Guard) Eval(rigID string, kind MotionKind) []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	var reasons []string
	mode := g.mode[rigID]
	if mode == model.DutyEmergency || mode == model.DutyFault {
		reasons = append(reasons, "duty inhibit")
	}
	if mode == model.DutyWindHold {
		reasons = append(reasons, "wind hold mode")
	}
	if err := g.windFault[rigID]; err != nil {
		if wind.IsBan(err) {
			reasons = append(reasons, "wind ban")
		} else if wind.IsRecoverable(err) && !g.windWindow.Closed(rigID) {
			reasons = append(reasons, "wind gust window")
		}
	}
	if err := g.loadFault[rigID]; err != nil {
		if load.IsMoment(err) {
			reasons = append(reasons, "moment exceeded")
		}
		if load.IsStale(err) {
			reasons = append(reasons, "stale load")
		}
	}
	if g.limits.SlewBoomMutex && kind == MotionSlew && g.locks[rigID][MotionLuff] {
		reasons = append(reasons, "luff active")
	}
	if g.limits.SlewBoomMutex && kind == MotionLuff && g.locks[rigID][MotionSlew] {
		reasons = append(reasons, "slew active")
	}
	return reasons
}

type RigInterlock struct {
	RigID, WindFault, LoadFault string
	Mode                        model.DutyMode
	WindHold                    bool
	LockedKinds                 []MotionKind
}

func (g *Guard) Snapshot() []RigInterlock {
	g.mu.Lock()
	defer g.mu.Unlock()
	var out []RigInterlock
	for rigID, mode := range g.mode {
		s := RigInterlock{RigID: rigID, Mode: mode, WindHold: g.windWindow.Active(rigID)}
		if err := g.windFault[rigID]; err != nil {
			s.WindFault = err.Error()
		}
		if err := g.loadFault[rigID]; err != nil {
			s.LoadFault = err.Error()
		}
		for k, v := range g.locks[rigID] {
			if v {
				s.LockedKinds = append(s.LockedKinds, k)
			}
		}
		out = append(out, s)
	}
	return out
}

func (g *Guard) StaleLoadCheck(rigID string) error {
	g.mu.Lock()
	err := g.loadFault[rigID]
	g.mu.Unlock()
	if load.IsStale(err) {
		return err
	}
	return nil
}

func Now() time.Time { return time.Now().UTC() }
