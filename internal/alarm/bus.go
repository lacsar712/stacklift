// Package alarm manages operator alarms for crane rigs.
package alarm

import (
	"fmt"
	"sync"
	"time"

	"github.com/lacsar712/stacklift/internal/model"
)

type Bus struct {
	mu      sync.Mutex
	active  map[string]model.Alarm
	history []model.Alarm
	hold    time.Duration
}

func NewBus(hold time.Duration) *Bus {
	return &Bus{active: make(map[string]model.Alarm), history: []model.Alarm{}, hold: hold}
}

func (b *Bus) Raise(rigID, code string, level model.AlarmLevel, message string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	key := rigID + ":" + code
	a := model.Alarm{RigID: rigID, Code: code, Level: level, Message: message, Since: time.Now().UTC()}
	b.active[key] = a
	b.history = append(b.history, a.Clone())
}

func (b *Bus) RaiseMoment(rigID string, pct float64) {
	b.Raise(rigID, "MOMENT_EXCEEDED", model.AlarmCritical, "moment: "+formatPct(pct))
}

func (b *Bus) RaiseWind(rigID, class string, speed float64) {
	level := model.AlarmWarn
	if class == "ban" {
		level = model.AlarmCritical
	}
	b.Raise(rigID, "WIND_"+class, level, "wind: "+formatPct(speed))
}

func (b *Bus) RaiseStaleLoad(rigID string) { b.Raise(rigID, "STALE_LOAD", model.AlarmWarn, "stale load") }
func (b *Bus) RaiseEmergency(rigID string) { b.Raise(rigID, "EMERGENCY", model.AlarmCritical, "emergency") }

func (b *Bus) ActiveSnapshot() []model.Alarm {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]model.Alarm, 0, len(b.active))
	for _, a := range b.active {
		out = append(out, a.Clone())
	}
	return out
}

func (b *Bus) HistorySnapshot() []model.Alarm {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]model.Alarm, len(b.history))
	copy(out, b.history)
	return out
}

func (b *Bus) CountActive() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.active)
}

func (b *Bus) HasCritical() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, a := range b.active {
		if a.Level == model.AlarmCritical {
			return true
		}
	}
	return false
}

func (b *Bus) Tick(now time.Time) {
	if b.hold <= 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for key, a := range b.active {
		if now.Sub(a.Since) > b.hold {
			delete(b.active, key)
		}
	}
}

func formatPct(v float64) string { return fmt.Sprintf("%.1f", v) }
