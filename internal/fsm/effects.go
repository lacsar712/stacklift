package fsm

import (
	"context"
	"fmt"

	"github.com/lacsar712/stacklift/internal/clock"
	"github.com/lacsar712/stacklift/internal/crane"
	"github.com/lacsar712/stacklift/internal/model"
)

func MotionSideEffects(coord *crane.Coordinator, clk *clock.DualClock, limits model.LimitSet) TransitionEffect {
	return func(ctx context.Context, craneID model.CraneID, from, to model.CranePhase) error {
		svc, ok := coord.Get(craneID)
		if !ok {
			return model.ErrCraneNotFound
		}
		switch to {
		case model.PhaseTraveling:
			svc.SetPhase(model.PhaseTraveling)
		case model.PhaseHoisting:
			svc.SetPhase(model.PhaseHoisting)
		case model.PhaseForking:
			svc.SetPhase(model.PhaseForking)
		case model.PhaseDwell:
			if m, ok := ctx.Value(dwellMachineKey{}).(*MotionMachine); ok {
				m.SetDwellTick(craneID, clk.ProcessTick())
			}
		case model.PhaseEmergency:
			svc.SetEStop(true)
			svc.CancelMotion()
		case model.PhaseIdle:
			if from == model.PhaseEmergency {
				svc.SetEStop(false)
			}
			svc.SetPhase(model.PhaseIdle)
		case model.PhaseFault:
			svc.CancelMotion()
			svc.SetPhase(model.PhaseFault)
		}
		if from == model.PhaseDwell {
			select {
			case <-ctx.Done():
				return fmt.Errorf("dwell check: %w", model.ErrContextDone)
			default:
			}
		}
		return nil
	}
}

type dwellMachineKey struct{}

func WithDwellMachine(ctx context.Context, m *MotionMachine) context.Context {
	return context.WithValue(ctx, dwellMachineKey{}, m)
}

func RegisterDefaultEffects(m *MotionMachine, coord *crane.Coordinator, clk *clock.DualClock, limits model.LimitSet) {
	m.OnTransition(MotionSideEffects(coord, clk, limits))
}

func EStopEffect(coord *crane.Coordinator) TransitionEffect {
	return func(ctx context.Context, craneID model.CraneID, from, to model.CranePhase) error {
		if to != model.PhaseEmergency {
			return nil
		}
		svc, ok := coord.Get(craneID)
		if !ok {
			return model.ErrCraneNotFound
		}
		svc.SetEStop(true)
		svc.CancelMotion()
		return nil
	}
}
