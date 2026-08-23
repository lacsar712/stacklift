package fork_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lacsar712/stacklift/internal/fork"
	"github.com/lacsar712/stacklift/internal/model"
)

func TestActuatorExtendRetract(t *testing.T) {
	act := fork.NewActuator("CR-01", 2500)
	ctx := context.Background()
	if err := act.Extend(ctx, 2000); err != nil {
		t.Fatal(err)
	}
	if !act.IsExtended() {
		t.Fatal("expected extended")
	}
	if err := act.Retract(ctx); err != nil {
		t.Fatal(err)
	}
	if !act.IsRetracted() {
		t.Fatal("expected retracted")
	}
}

func TestActuatorContextCancel(t *testing.T) {
	act := fork.NewActuator("CR-01", 2500)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := act.Extend(ctx, 1000); err == nil {
		t.Fatal("expected cancel error")
	} else if !errors.Is(err, model.ErrContextDone) {
		t.Fatalf("expected ErrContextDone got %v", err)
	}
}

func TestTelescope(t *testing.T) {
	tel := fork.NewTelescope("CR-01", 2500)
	ctx := context.Background()
	if err := tel.ExtendBoth(ctx, 2000); err != nil {
		t.Fatal(err)
	}
	if err := tel.RetractBoth(ctx); err != nil {
		t.Fatal(err)
	}
	if !tel.BothRetracted() {
		t.Fatal("expected both retracted")
	}
}

func TestLoadHandler(t *testing.T) {
	h := fork.NewLoadHandler("CR-01", 2500)
	ctx := context.Background()
	if err := h.Pickup(ctx, 1500); err != nil {
		t.Fatal(err)
	}
	if err := h.Deposit(ctx, 1500); err != nil {
		t.Fatal(err)
	}
}
