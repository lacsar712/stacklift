package app

import (
	"context"
	"testing"

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
	craneID := a.Coordinator().CraneIDs()[0]
	aisleID, _ := a.Registry().AisleForCrane(craneID)
	bad := model.Location{Aisle: aisleID, Bay: 1, Level: 1, Depth: 0}
	good := model.Location{Aisle: aisleID, Bay: 3, Level: 1, Depth: 0}
	svc, _ := a.Coordinator().Get(craneID)
	svc.SetInterlocked(true)
	_ = a.MoveCrane(context.Background(), craneID, bad, good)
	svc.SetInterlocked(false)
	if err := a.MoveCrane(context.Background(), craneID, bad, good); err != nil {
		t.Fatalf("second move blocked by hoist lease: %v", err)
	}
}
