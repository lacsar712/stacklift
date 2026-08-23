package app

import (
	"context"
	"testing"

	"github.com/lacsar712/stacklift/internal/config"
	"github.com/lacsar712/stacklift/internal/model"
)

func TestCase(t *testing.T) {
	a, err := NewDefault()
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	cranes := a.Coordinator().CraneIDs()
	if len(cranes) == 0 {
		t.Fatal("no cranes")
	}
	craneID := cranes[0]
	aisleID, _ := a.Registry().AisleForCrane(craneID)
	layout := config.NewLayout(a.Config().Warehouse)
	source := model.Location{Aisle: aisleID, Bay: 2, Level: 1, Depth: 1}
	dest := model.Location{Aisle: aisleID, Bay: 4, Level: 2, Depth: 1}
	_ = a.Occupancy().Place(source, model.PalletID("PLT-X"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	before, _ := a.Coordinator().Get(craneID)
	beforeMM := before.ForkHandler().Telescope().Left().ExtensionMM()
	if err := a.RetrievePallet(ctx, craneID, model.PalletID("PLT-X"), dest); err == nil {
		after, _ := a.Coordinator().Get(craneID)
		if after.ForkHandler().Telescope().Left().ExtensionMM() > beforeMM {
			t.Fatal("fork extended after cancel")
		}
	}
}
