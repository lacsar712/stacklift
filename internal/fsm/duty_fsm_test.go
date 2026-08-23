package fsm_test

import (
	"context"
	"testing"

	"github.com/lacsar712/stacklift/internal/fsm"
	"github.com/lacsar712/stacklift/internal/model"
	"github.com/lacsar712/stacklift/internal/slew"
)

func TestTransition(t *testing.T) {
	m := fsm.NewDutyMachine(slew.NewEmitter())
	m.Ensure("TC-01")
	res, err := m.Transition(context.Background(), model.TransitionRequest{RigID: "TC-01", To: model.DutySlew, Tick: 1})
	if err != nil || !res.Accepted {
		t.Fatal(err)
	}
}

func TestIllegal(t *testing.T) {
	m := fsm.NewDutyMachine(slew.NewEmitter())
	m.Ensure("TC-01")
	_, _ = m.Transition(context.Background(), model.TransitionRequest{RigID: "TC-01", To: model.DutyEmergency, Tick: 1})
	_, err := m.Transition(context.Background(), model.TransitionRequest{RigID: "TC-01", To: model.DutySlew, Tick: 2})
	if err == nil {
		t.Fatal("illegal")
	}
}
