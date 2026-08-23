package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lacsar712/stacklift/internal/app"
	"github.com/lacsar712/stacklift/internal/model"
)

func TestNewDefault(t *testing.T) {
	a, err := app.NewDefault()
	if err != nil || a == nil {
		t.Fatalf("new default: %v", err)
	}
}

func TestStart(t *testing.T) {
	a, _ := app.NewDefault()
	if err := a.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !a.Started() || a.CraneCount() != 4 {
		t.Fatalf("started=%v cranes=%d", a.Started(), a.CraneCount())
	}
}

func TestSnapshot(t *testing.T) {
	a, _ := app.NewDefault()
	_ = a.Start(context.Background())
	if len(a.Snapshot().Cranes) == 0 {
		t.Fatal("empty snapshot")
	}
}

func TestMoveCrane(t *testing.T) {
	a, _ := app.NewDefault()
	_ = a.Start(context.Background())
	from := model.Location{Aisle: "01", Bay: 1, Level: 1, Depth: 0}
	to := model.Location{Aisle: "01", Bay: 2, Level: 1, Depth: 0}
	if err := a.MoveCrane(context.Background(), "CR-01", from, to); err != nil {
		t.Fatal(err)
	}
}

func TestRunDemo(t *testing.T) {
	a, _ := app.NewDefault()
	if err := app.RunDemo(context.Background(), a); err != nil {
		t.Fatal(err)
	}
}

func TestPlacePallet(t *testing.T) {
	a, _ := app.NewDefault()
	_ = a.Start(context.Background())
	loc := model.Location{Aisle: "01", Bay: 1, Level: 1, Depth: 0}
	if err := a.PlacePallet(loc, "PLT-1"); err != nil {
		t.Fatal(err)
	}
}

func TestRetrievePallet(t *testing.T) {
	a, _ := app.NewDefault()
	ctx := context.Background()
	_ = a.Start(ctx)
	source := model.Location{Aisle: "01", Bay: 2, Level: 1, Depth: 0}
	dest := model.Location{Aisle: "01", Bay: 4, Level: 2, Depth: 0}
	_ = a.PlacePallet(source, "PLT-RET")
	if err := a.RetrievePallet(ctx, "CR-01", "PLT-RET", dest); err != nil {
		t.Fatal(err)
	}
}

func TestAdvanceClock(t *testing.T) {
	a, _ := app.NewDefault()
	if a.AdvanceClock(10) != 10 {
		t.Fatal("expected tick 10")
	}
}

func TestMoveCraneNotFound(t *testing.T) {
	a, _ := app.NewDefault()
	_ = a.Start(context.Background())
	from := model.Location{Aisle: "01", Bay: 1, Level: 1, Depth: 0}
	to := model.Location{Aisle: "01", Bay: 2, Level: 1, Depth: 0}
	if !errors.Is(a.MoveCrane(context.Background(), "CR-99", from, to), model.ErrCraneNotFound) {
		t.Fatal("expected not found")
	}
}
