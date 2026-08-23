package crane_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lacsar712/stacklift/internal/clock"
	"github.com/lacsar712/stacklift/internal/crane"
	"github.com/lacsar712/stacklift/internal/model"
)

func TestCase(t *testing.T) {
	clk := clock.NewDual(100)
	svc := crane.NewService("CR-01", model.DefaultLimits(), clk)
	err := svc.MoveAxis(context.Background(), model.AxisTravel, 10000)
	if err == nil {
		t.Fatal("expected encoder slip")
	}
	wrapped := crane.WrapMotionFault(err)
	if !errors.Is(wrapped, model.ErrEncoderSlip) {
		t.Fatalf("expected ErrEncoderSlip, got %v", wrapped)
	}
}
