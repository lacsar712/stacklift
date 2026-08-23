package model

type CraneEvent string

const (
	EventStartTravel  CraneEvent = "start_travel"
	EventTravelDone   CraneEvent = "travel_done"
	EventStartHoist   CraneEvent = "start_hoist"
	EventHoistDone    CraneEvent = "hoist_done"
	EventStartFork    CraneEvent = "start_fork"
	EventForkDone     CraneEvent = "fork_done"
	EventLoadPallet   CraneEvent = "load_pallet"
	EventUnloadPallet CraneEvent = "unload_pallet"
	EventDwellStart   CraneEvent = "dwell_start"
	EventDwellEnd     CraneEvent = "dwell_end"
	EventEStop        CraneEvent = "estop"
	EventReset        CraneEvent = "reset"
	EventFault        CraneEvent = "fault"
)

var phaseTransitions = map[CranePhase]map[CraneEvent]CranePhase{
	PhaseIdle: {
		EventStartTravel: PhaseTraveling, EventStartHoist: PhaseHoisting,
		EventStartFork: PhaseForking, EventEStop: PhaseEmergency, EventFault: PhaseFault,
	},
	PhaseTraveling: {EventTravelDone: PhaseIdle, EventEStop: PhaseEmergency, EventFault: PhaseFault},
	PhaseHoisting:  {EventHoistDone: PhaseIdle, EventEStop: PhaseEmergency, EventFault: PhaseFault},
	PhaseForking:   {EventForkDone: PhaseIdle, EventEStop: PhaseEmergency, EventFault: PhaseFault},
	PhaseLoading:   {EventLoadPallet: PhaseDwell, EventEStop: PhaseEmergency, EventFault: PhaseFault},
	PhaseUnloading: {EventUnloadPallet: PhaseDwell, EventEStop: PhaseEmergency, EventFault: PhaseFault},
	PhaseDwell:     {EventDwellEnd: PhaseIdle, EventEStop: PhaseEmergency, EventFault: PhaseFault},
	PhaseEmergency: {EventReset: PhaseIdle, EventFault: PhaseFault},
	PhaseFault:     {EventReset: PhaseIdle},
}

func NextPhase(current CranePhase, event CraneEvent) (CranePhase, bool) {
	events, ok := phaseTransitions[current]
	if !ok {
		return current, false
	}
	next, ok := events[event]
	return next, ok
}

func ActiveAxis(phase CranePhase) (MotionAxis, bool) {
	switch phase {
	case PhaseTraveling:
		return AxisTravel, true
	case PhaseHoisting:
		return AxisHoist, true
	case PhaseForking:
		return AxisFork, true
	default:
		return "", false
	}
}

func IsMotionPhase(phase CranePhase) bool {
	switch phase {
	case PhaseTraveling, PhaseHoisting, PhaseForking, PhaseLoading, PhaseUnloading:
		return true
	default:
		return false
	}
}

func EmptyCraneStatus(id CraneID) CraneStatus {
	return CraneStatus{CraneID: id, Phase: PhaseIdle, Pose: CranePose{CraneID: id, ForkPos: ForkRetracted}, Revision: 1}
}
