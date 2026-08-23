package fsm

import (
	"context"
	"fmt"
	"sync"

	"github.com/lacsar712/stacklift/internal/model"
)

var allowed = map[model.CranePhase]map[model.CraneEvent]model.CranePhase{
	model.PhaseIdle: {
		model.EventStartTravel: model.PhaseTraveling, model.EventStartHoist: model.PhaseHoisting,
		model.EventStartFork: model.PhaseForking, model.EventEStop: model.PhaseEmergency, model.EventFault: model.PhaseFault,
	},
	model.PhaseTraveling: {model.EventTravelDone: model.PhaseIdle, model.EventEStop: model.PhaseEmergency, model.EventFault: model.PhaseFault},
	model.PhaseHoisting:  {model.EventHoistDone: model.PhaseIdle, model.EventEStop: model.PhaseEmergency, model.EventFault: model.PhaseFault},
	model.PhaseForking:   {model.EventForkDone: model.PhaseIdle, model.EventEStop: model.PhaseEmergency, model.EventFault: model.PhaseFault},
	model.PhaseLoading:   {model.EventLoadPallet: model.PhaseDwell, model.EventEStop: model.PhaseEmergency, model.EventFault: model.PhaseFault},
	model.PhaseUnloading: {model.EventUnloadPallet: model.PhaseDwell, model.EventEStop: model.PhaseEmergency, model.EventFault: model.PhaseFault},
	model.PhaseDwell:     {model.EventDwellEnd: model.PhaseIdle, model.EventEStop: model.PhaseEmergency, model.EventFault: model.PhaseFault},
	model.PhaseEmergency: {model.EventReset: model.PhaseIdle, model.EventFault: model.PhaseFault},
	model.PhaseFault:     {model.EventReset: model.PhaseIdle},
}

func CanTransition(from model.CranePhase, event model.CraneEvent) bool {
	events, ok := allowed[from]
	if !ok {
		return false
	}
	_, ok = events[event]
	return ok
}

func TargetPhase(from model.CranePhase, event model.CraneEvent) (model.CranePhase, bool) {
	events, ok := allowed[from]
	if !ok {
		return from, false
	}
	to, ok := events[event]
	return to, ok
}

type TransitionEffect func(ctx context.Context, crane model.CraneID, from, to model.CranePhase) error

type MotionMachine struct {
	mu        sync.Mutex
	phases    map[model.CraneID]model.CranePhase
	revision  map[model.CraneID]uint64
	effects   []TransitionEffect
	dwellTick map[model.CraneID]int64
}

func NewMotionMachine() *MotionMachine {
	return &MotionMachine{
		phases: make(map[model.CraneID]model.CranePhase), revision: make(map[model.CraneID]uint64),
		dwellTick: make(map[model.CraneID]int64),
	}
}

func (m *MotionMachine) OnTransition(effect TransitionEffect) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.effects = append(m.effects, effect)
}

func (m *MotionMachine) Ensure(crane model.CraneID) model.CranePhase {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ensureLocked(crane)
}

func (m *MotionMachine) ensureLocked(crane model.CraneID) model.CranePhase {
	if p, ok := m.phases[crane]; ok {
		return p
	}
	m.phases[crane] = model.PhaseIdle
	m.revision[crane] = 1
	return model.PhaseIdle
}

func (m *MotionMachine) PhaseOf(crane model.CraneID) model.CranePhase {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ensureLocked(crane)
}

func (m *MotionMachine) Fire(ctx context.Context, crane model.CraneID, event model.CraneEvent) (model.TransitionResult, error) {
	m.mu.Lock()
	from := m.ensureLocked(crane)
	to, ok := TargetPhase(from, event)
	if !ok {
		rev := m.revision[crane]
		m.mu.Unlock()
		return model.TransitionResult{false, from, "illegal", rev}, model.NewPhaseError(from, from, model.ErrInvalidPhase)
	}
	if from == to {
		rev := m.revision[crane]
		m.mu.Unlock()
		return model.TransitionResult{true, from, "noop", rev}, nil
	}
	m.phases[crane] = to
	m.revision[crane]++
	rev := m.revision[crane]
	effects := append([]TransitionEffect{}, m.effects...)
	m.mu.Unlock()
	for _, fx := range effects {
		if err := fx(ctx, crane, from, to); err != nil {
			return model.TransitionResult{false, from, err.Error(), rev}, err
		}
	}
	return model.TransitionResult{true, to, "ok", rev}, nil
}

func (m *MotionMachine) Transition(ctx context.Context, req model.TransitionRequest) (model.TransitionResult, error) {
	m.mu.Lock()
	from := m.ensureLocked(req.CraneID)
	if req.From != "" && req.From != from {
		rev := m.revision[req.CraneID]
		m.mu.Unlock()
		return model.TransitionResult{false, from, "from mismatch", rev}, fmt.Errorf("fsm from mismatch")
	}
	to := req.To
	if from == to {
		rev := m.revision[req.CraneID]
		m.mu.Unlock()
		return model.TransitionResult{true, from, "noop", rev}, nil
	}
	if !req.Force {
		found := false
		for _, target := range allowed[from] {
			if target == to {
				found = true
				break
			}
		}
		if !found {
			rev := m.revision[req.CraneID]
			m.mu.Unlock()
			return model.TransitionResult{false, from, "illegal", rev}, model.NewPhaseError(from, to, model.ErrInvalidPhase)
		}
	}
	m.phases[req.CraneID] = to
	m.revision[req.CraneID]++
	rev := m.revision[req.CraneID]
	effects := append([]TransitionEffect{}, m.effects...)
	m.mu.Unlock()
	for _, fx := range effects {
		if err := fx(ctx, req.CraneID, from, to); err != nil {
			return model.TransitionResult{false, from, err.Error(), rev}, err
		}
	}
	return model.TransitionResult{true, to, "ok", rev}, nil
}

func (m *MotionMachine) SetDwellTick(crane model.CraneID, tick int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dwellTick[crane] = tick
}

func (m *MotionMachine) DwellTick(crane model.CraneID) (int64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.dwellTick[crane]
	return t, ok
}

func (m *MotionMachine) Snapshot(crane model.CraneID) (model.CranePhase, uint64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.phases[crane]
	if !ok {
		return model.PhaseIdle, 0, false
	}
	return p, m.revision[crane], true
}

func (m *MotionMachine) SnapshotAll() map[model.CraneID]model.CranePhase {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[model.CraneID]model.CranePhase, len(m.phases))
	for id, p := range m.phases {
		out[id] = p
	}
	return out
}
