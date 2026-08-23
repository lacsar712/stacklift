package fsm

import (
	"context"
	"fmt"
	"sync"

	"github.com/lacsar712/stacklift/internal/model"
	"github.com/lacsar712/stacklift/internal/slew"
)

var allowed = map[model.DutyMode]map[model.DutyMode]bool{
	model.DutyIdle: {model.DutyHoist: true, model.DutySlew: true, model.DutyLuff: true, model.DutyEmergency: true, model.DutyWindHold: true, model.DutyFault: true},
	model.DutyHoist: {model.DutyIdle: true, model.DutyEmergency: true, model.DutyWindHold: true, model.DutyFault: true},
	model.DutySlew:  {model.DutyIdle: true, model.DutyEmergency: true, model.DutyWindHold: true, model.DutyFault: true},
	model.DutyLuff:  {model.DutyIdle: true, model.DutyEmergency: true, model.DutyWindHold: true, model.DutyFault: true},
	model.DutyWindHold: {model.DutyIdle: true, model.DutyEmergency: true, model.DutyFault: true},
	model.DutyEmergency: {model.DutyIdle: true, model.DutyFault: true},
	model.DutyFault: {model.DutyIdle: true},
}

func CanTransition(from, to model.DutyMode) bool {
	if from == to {
		return true
	}
	next, ok := allowed[from]
	return ok && next[to]
}

type DutyMachine struct {
	mu       sync.Mutex
	modes    map[string]model.DutyMode
	revision map[string]uint64
	emitter  *slew.Emitter
}

func NewDutyMachine(emitter *slew.Emitter) *DutyMachine {
	return &DutyMachine{modes: make(map[string]model.DutyMode), revision: make(map[string]uint64), emitter: emitter}
}

func (d *DutyMachine) Ensure(rigID string) model.DutyMode {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.ensureLocked(rigID)
}

func (d *DutyMachine) ModeOf(rigID string) model.DutyMode {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.ensureLocked(rigID)
}

func (d *DutyMachine) ensureLocked(rigID string) model.DutyMode {
	if m, ok := d.modes[rigID]; ok {
		return m
	}
	d.modes[rigID] = model.DutyIdle
	d.revision[rigID] = 1
	return model.DutyIdle
}

func (d *DutyMachine) Transition(ctx context.Context, req model.TransitionRequest) (model.TransitionResult, error) {
	d.mu.Lock()
	from := d.ensureLocked(req.RigID)
	if req.From != "" && req.From != from {
		rev := d.revision[req.RigID]
		d.mu.Unlock()
		return model.TransitionResult{false, from, "from mismatch", rev}, fmt.Errorf("fsm from mismatch")
	}
	to := req.To
	if from == to {
		rev := d.revision[req.RigID]
		d.mu.Unlock()
		return model.TransitionResult{true, from, "noop", rev}, nil
	}
	if !req.Force && !CanTransition(from, to) {
		rev := d.revision[req.RigID]
		d.mu.Unlock()
		return model.TransitionResult{false, from, "illegal", rev}, fmt.Errorf("illegal transition")
	}
	d.modes[req.RigID] = to
	d.revision[req.RigID]++
	rev := d.revision[req.RigID]
	d.mu.Unlock()
	if to == model.DutyEmergency || to == model.DutySlew || to == model.DutyLuff {
		if err := d.emitter.Emit(ctx, req.RigID, d.emitter.AngleOf(req.RigID)); err != nil {
			return model.TransitionResult{false, from, err.Error(), rev}, err
		}
	}
	return model.TransitionResult{true, to, "ok", rev}, nil
}

func (d *DutyMachine) Snapshot(rigID string) (model.DutyMode, uint64, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	m, ok := d.modes[rigID]
	if !ok {
		return model.DutyIdle, 0, false
	}
	return m, d.revision[rigID], true
}

func (d *DutyMachine) SnapshotAll() map[string]model.DutyMode {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make(map[string]model.DutyMode, len(d.modes))
	for id, m := range d.modes {
		out[id] = m
	}
	return out
}
