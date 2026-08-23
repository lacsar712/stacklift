package model

import "errors"

var (
	ErrContextDone      = errors.New("operation cancelled")
	ErrInterlockBlocked = errors.New("interlock blocked")
	ErrInvalidLocation  = errors.New("invalid location")
	ErrCellOccupied     = errors.New("cell occupied")
	ErrCellEmpty        = errors.New("cell empty")
	ErrCellReserved     = errors.New("cell reserved")
	ErrMotionInProgress = errors.New("motion in progress")
	ErrHoistLocked      = errors.New("hoist locked")
	ErrEStopActive      = errors.New("emergency stop active")
	ErrInvalidPhase     = errors.New("invalid phase transition")
	ErrCraneNotFound    = errors.New("crane not found")
	ErrTaskNotFound     = errors.New("task not found")
	ErrSnapshotStale    = errors.New("snapshot stale")
	ErrAxisConflict        = errors.New("axis conflict")
	ErrTravelOverspeed     = errors.New("travel overspeed")
	ErrEncoderSlip         = errors.New("encoder slip")
	ErrForkLoadImbalance   = errors.New("fork load imbalance")
)

type MotionError struct {
	Axis    MotionAxis
	CraneID CraneID
	Cause   error
}

func (e *MotionError) Error() string {
	return "motion " + string(e.Axis) + " crane " + string(e.CraneID) + ": " + e.Cause.Error()
}

func (e *MotionError) Unwrap() error { return e.Cause }

func NewMotionError(axis MotionAxis, crane CraneID, cause error) error {
	return &MotionError{Axis: axis, CraneID: crane, Cause: cause}
}

type InterlockError struct {
	Rule    string
	CraneID CraneID
	Detail  string
}

func (e *InterlockError) Error() string {
	return "interlock " + e.Rule + " crane " + string(e.CraneID) + ": " + e.Detail
}

func (e *InterlockError) Unwrap() error { return ErrInterlockBlocked }

func NewInterlockError(rule string, crane CraneID, detail string) error {
	return &InterlockError{Rule: rule, CraneID: crane, Detail: detail}
}

type PhaseError struct {
	From  CranePhase
	To    CranePhase
	Cause error
}

func (e *PhaseError) Error() string {
	return "phase " + string(e.From) + "->" + string(e.To) + ": " + e.Cause.Error()
}

func (e *PhaseError) Unwrap() error { return e.Cause }

func NewPhaseError(from, to CranePhase, cause error) error {
	return &PhaseError{From: from, To: to, Cause: cause}
}
