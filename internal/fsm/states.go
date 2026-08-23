package fsm

import "github.com/lacsar712/stacklift/internal/model"

type StateTable struct {
	entries map[string]Entry
}

type Entry struct {
	RigID    string
	Mode     model.DutyMode
	Revision uint64
}

func BuildStateTable(machine *DutyMachine) StateTable {
	modes := machine.SnapshotAll()
	entries := make(map[string]Entry, len(modes))
	for id, mode := range modes {
		_, rev, _ := machine.Snapshot(id)
		entries[id] = Entry{id, mode, rev}
	}
	return StateTable{entries}
}

func (t StateTable) Count() int { return len(t.entries) }

func (t StateTable) IsEmergency() bool {
	for _, e := range t.entries {
		if e.Mode == model.DutyEmergency {
			return true
		}
	}
	return false
}
