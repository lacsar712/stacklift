package store

import (
	"sync"

	"github.com/lacsar712/stacklift/internal/aisle"
	"github.com/lacsar712/stacklift/internal/crane"
	"github.com/lacsar712/stacklift/internal/clock"
	"github.com/lacsar712/stacklift/internal/model"
	"github.com/lacsar712/stacklift/internal/safety"
)

type Rig struct {
	mu        sync.RWMutex
	coord     *crane.Coordinator
	occupancy *aisle.Occupancy
	monitor   *safety.Monitor
	snapshots *SnapshotStore
	clk       *clock.DualClock
}

func NewRig(coord *crane.Coordinator, occ *aisle.Occupancy, monitor *safety.Monitor, clk *clock.DualClock) *Rig {
	return &Rig{coord: coord, occupancy: occ, monitor: monitor, snapshots: NewSnapshotStore(), clk: clk}
}

func (r *Rig) Capture() model.WarehouseSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	snap := model.NewWarehouseSnapshot(r.snapshots.Revision()+1, r.clk.ProcessTick())
	snap.Cranes = r.coord.SnapshotAll()
	snap.Cells = r.occupancy.Snapshot()
	snap.Alarms = r.monitor.Alarms()
	r.snapshots.Save(snap)
	return snap.Clone()
}

func (r *Rig) SnapshotStore() *SnapshotStore { return r.snapshots }
func (r *Rig) Latest() model.WarehouseSnapshot { return r.snapshots.Load() }

func (r *Rig) RefreshCrane(id model.CraneID) {
	svc, ok := r.coord.Get(id)
	if !ok {
		return
	}
	r.snapshots.UpdateCrane(id, svc.Status())
}

func (r *Rig) RefreshCells() { r.snapshots.UpdateCells(r.occupancy.Snapshot()) }

func (r *Rig) Coordinator() *crane.Coordinator { return r.coord }
func (r *Rig) Occupancy() *aisle.Occupancy     { return r.occupancy }
func (r *Rig) Monitor() *safety.Monitor        { return r.monitor }
func (r *Rig) Clock() *clock.DualClock         { return r.clk }
func (r *Rig) CraneCount() int                 { return r.coord.Count() }
func (r *Rig) OccupiedCells() int              { return r.occupancy.CountOccupied() }
