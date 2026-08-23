package aisle_test

import (
	"testing"

	"github.com/lacsar712/stacklift/internal/aisle"
	"github.com/lacsar712/stacklift/internal/config"
	"github.com/lacsar712/stacklift/internal/model"
)

func TestRegistry(t *testing.T) {
	reg := aisle.NewRegistry()
	info := aisle.AisleInfo{ID: "01", CraneID: "CR-01", LengthMM: 50000, Active: true}
	if err := reg.Register(info); err != nil {
		t.Fatal(err)
	}
	crane, ok := reg.CraneForAisle("01")
	if !ok || crane != "CR-01" {
		t.Fatal("crane for aisle failed")
	}
}

func TestOccupancy(t *testing.T) {
	layout := config.NewLayout(config.DefaultWarehouse())
	occ := aisle.NewOccupancy(layout.AllCells())
	loc := model.Location{Aisle: "01", Bay: 1, Level: 1, Depth: 0}
	if err := occ.Place(loc, "PLT-1"); err != nil {
		t.Fatal(err)
	}
	if !occ.IsOccupied(loc) {
		t.Fatal("cell should be occupied")
	}
	if err := occ.Remove(loc); err != nil {
		t.Fatal(err)
	}
}

func TestReserve(t *testing.T) {
	layout := config.NewLayout(config.DefaultWarehouse())
	occ := aisle.NewOccupancy(layout.AllCells())
	loc := model.Location{Aisle: "01", Bay: 2, Level: 1, Depth: 0}
	if err := occ.Reserve(loc, "CR-01"); err != nil {
		t.Fatal(err)
	}
	if err := occ.Place(loc, "PLT-2"); err == nil {
		t.Fatal("expected reserved error")
	}
}

func TestFindPallet(t *testing.T) {
	layout := config.NewLayout(config.DefaultWarehouse())
	occ := aisle.NewOccupancy(layout.AllCells())
	loc := model.Location{Aisle: "01", Bay: 3, Level: 2, Depth: 0}
	_ = occ.Place(loc, "PLT-FIND")
	found, ok := occ.FindPallet("PLT-FIND")
	if !ok || !found.Equal(loc) {
		t.Fatalf("find failed: %v", found)
	}
}

func TestSnapshotClone(t *testing.T) {
	layout := config.NewLayout(config.DefaultWarehouse())
	occ := aisle.NewOccupancy(layout.AllCells())
	loc := model.Location{Aisle: "01", Bay: 1, Level: 1, Depth: 0}
	_ = occ.Place(loc, "PLT-SNAP")
	snap := occ.Snapshot()
	snap[0].Occupied = false
	if !occ.IsOccupied(loc) {
		t.Fatal("snapshot mutation affected store")
	}
}
