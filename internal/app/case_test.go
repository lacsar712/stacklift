package app

import (
	"context"
	"errors"
	"testing"

	"github.com/lacsar712/stacklift/internal/config"
	"github.com/lacsar712/stacklift/internal/model"
)

func TestCase(t *testing.T) {
	cfg := config.Default()
	cfg.Limits.MaxTravelSpeed = 100
	a, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	craneID := a.Coordinator().CraneIDs()[0]
	aisleID, _ := a.Registry().AisleForCrane(craneID)
	from := model.Location{Aisle: aisleID, Bay: 1, Level: 1, Depth: 0}
	to := model.Location{Aisle: aisleID, Bay: 10, Level: 1, Depth: 0}
	err = a.MoveCrane(context.Background(), craneID, from, to)
	if err == nil {
		t.Fatal("expected overspeed error")
	}
	if !errors.Is(err, model.ErrTravelOverspeed) {
		t.Fatalf("expected ErrTravelOverspeed, got %v", err)
	}
}
