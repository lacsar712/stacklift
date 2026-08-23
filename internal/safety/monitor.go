package safety

import (
	"fmt"
	"sync"
	"time"

	"github.com/lacsar712/stacklift/internal/model"
)

type Monitor struct {
	mu     sync.RWMutex
	alarms []model.Alarm
	estop  bool
	limits model.LimitSet
}

func NewMonitor(limits model.LimitSet) *Monitor {
	return &Monitor{limits: limits}
}

func (m *Monitor) Raise(crane model.CraneID, code string, level model.AlarmLevel, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alarms = append(m.alarms, model.Alarm{CraneID: crane, Code: code, Level: level, Message: message, Since: time.Now()})
}

func (m *Monitor) Alarms() []model.Alarm {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]model.Alarm, len(m.alarms))
	for i, a := range m.alarms {
		out[i] = a.Clone()
	}
	return out
}

func (m *Monitor) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alarms = nil
}

func (m *Monitor) CheckPose(crane model.CraneID, pose model.CranePose) []model.Alarm {
	var raised []model.Alarm
	if pose.TravelMM > m.limits.MaxTravelMM {
		a := model.Alarm{CraneID: crane, Code: "TRAVEL_LIMIT", Level: model.AlarmCritical, Message: "travel position exceeds limit", Since: time.Now()}
		raised = append(raised, a)
		m.Raise(crane, a.Code, a.Level, a.Message)
	}
	if pose.HoistMM > m.limits.MaxHoistMM {
		a := model.Alarm{CraneID: crane, Code: "HOIST_LIMIT", Level: model.AlarmCritical, Message: "hoist position exceeds limit", Since: time.Now()}
		raised = append(raised, a)
		m.Raise(crane, a.Code, a.Level, a.Message)
	}
	if pose.ForkMM > m.limits.MaxForkMM {
		a := model.Alarm{CraneID: crane, Code: "FORK_LIMIT", Level: model.AlarmCritical, Message: "fork extension exceeds limit", Since: time.Now()}
		raised = append(raised, a)
		m.Raise(crane, a.Code, a.Level, a.Message)
	}
	return raised
}

func (m *Monitor) EStopActive() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.estop
}

func (m *Monitor) SetEStop(active bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.estop = active
}

func (m *Monitor) AlarmCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.alarms)
}

func (m *Monitor) CheckTravelSpeed(crane model.CraneID, speedMMS float64) error {
	if speedMMS > m.limits.MaxTravelSpeed {
		return fmt.Errorf("crane %s: %w", crane, model.ErrTravelOverspeed)
	}
	return nil
}

func (m *Monitor) CriticalCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := 0
	for _, a := range m.alarms {
		if a.Level == model.AlarmCritical {
			n++
		}
	}
	return n
}
