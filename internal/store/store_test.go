package store_test

import (
	"testing"

	"github.com/lacsar712/stacklift/internal/aisle"
	"github.com/lacsar712/stacklift/internal/clock"
	"github.com/lacsar712/stacklift/internal/config"
	"github.com/lacsar712/stacklift/internal/crane"
	"github.com/lacsar712/stacklift/internal/model"
	"github.com/lacsar712/stacklift/internal/path"
	"github.com/lacsar712/stacklift/internal/safety"
	"github.com/lacsar712/stacklift/internal/store"
)

func setup() *store.Rig {
	clk := clock.NewDual(100)
	layout := config.NewLayout(config.DefaultWarehouse())
	planner := path.NewPlanner(layout, model.DefaultLimits())
	coord := crane.NewCoordinator(planner, clk, model.DefaultLimits())
	coord.Register("CR-01")
	occ := aisle.NewOccupancy(layout.AllCells())
	monitor := safety.NewMonitor(model.DefaultLimits())
	return store.NewRig(coord, occ, monitor, clk)
}

func TestSnapshotCloneProtection(t *testing.T) {
	ss := store.NewSnapshotStore()
	snap := model.NewWarehouseSnapshot(1, 0)
	snap.Cranes = map[model.CraneID]model.CraneStatus{"CR-01": {CraneID: "CR-01", Phase: model.PhaseIdle}}
	ss.Save(snap)
	loaded := ss.Load()
	loaded.Cranes["CR-01"] = model.CraneStatus{CraneID: "CR-01", Phase: model.PhaseFault}
	original, _ := ss.Crane("CR-01")
	if original.Phase == model.PhaseFault {
		t.Fatal("mutation of loaded snapshot affected store")
	}
}

func TestUpdateCrane(t *testing.T) {
	ss := store.NewSnapshotStore()
	ss.UpdateCrane("CR-01", model.EmptyCraneStatus("CR-01"))
	if ss.Revision() == 0 {
		t.Fatal("revision should increment")
	}
}

func TestCompareAndSwap(t *testing.T) {
	ss := store.NewSnapshotStore()
	ss.Save(model.NewWarehouseSnapshot(0, 0))
	rev := ss.Revision()
	if !ss.CompareAndSwap(rev, model.NewWarehouseSnapshot(rev, 5)) {
		t.Fatal("CAS should succeed")
	}
	if ss.CompareAndSwap(rev, model.NewWarehouseSnapshot(rev, 5)) {
		t.Fatal("stale CAS should fail")
	}
}

func TestRigCapture(t *testing.T) {
	rig := setup()
	snap := rig.Capture()
	if len(snap.Cranes) != 1 {
		t.Fatalf("expected 1 crane got %d", len(snap.Cranes))
	}
}

func TestIsStale(t *testing.T) {
	ss := store.NewSnapshotStore()
	ss.Save(model.NewWarehouseSnapshot(0, 0))
	if !ss.IsStale(0) {
		t.Fatal("seq 0 should be stale after save")
	}
}

func TestRigRefresh(t *testing.T) {
	rig := setup()
	rig.RefreshCrane("CR-01")
	if _, ok := rig.SnapshotStore().Crane("CR-01"); !ok {
		t.Fatal("crane not refreshed")
	}
}
