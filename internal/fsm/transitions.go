package fsm

import "github.com/lacsar712/stacklift/internal/model"

func EventForAxis(axis model.MotionAxis, starting bool) model.CraneEvent {
	if starting {
		switch axis {
		case model.AxisTravel:
			return model.EventStartTravel
		case model.AxisHoist:
			return model.EventStartHoist
		case model.AxisFork:
			return model.EventStartFork
		}
	}
	switch axis {
	case model.AxisTravel:
		return model.EventTravelDone
	case model.AxisHoist:
		return model.EventHoistDone
	case model.AxisFork:
		return model.EventForkDone
	default:
		return ""
	}
}

func PhaseForAxis(axis model.MotionAxis) model.CranePhase {
	switch axis {
	case model.AxisTravel:
		return model.PhaseTraveling
	case model.AxisHoist:
		return model.PhaseHoisting
	case model.AxisFork:
		return model.PhaseForking
	default:
		return model.PhaseIdle
	}
}

func IsTerminalPhase(phase model.CranePhase) bool {
	return phase == model.PhaseEmergency || phase == model.PhaseFault
}

func AllowedEvents(phase model.CranePhase) []model.CraneEvent {
	events, ok := allowed[phase]
	if !ok {
		return nil
	}
	out := make([]model.CraneEvent, 0, len(events))
	for ev := range events {
		out = append(out, ev)
	}
	return out
}

func CanRecover(phase model.CranePhase) bool {
	return phase == model.PhaseEmergency || phase == model.PhaseFault
}
