// Package model defines ASRS stacker crane domain types.
package model

import (
	"fmt"
	"time"
)

type CraneID string
type AisleID string
type BayID string
type PalletID string
type TaskID string

type MotionAxis string

const (
	AxisTravel MotionAxis = "travel"
	AxisHoist  MotionAxis = "hoist"
	AxisFork   MotionAxis = "fork"
)

type CranePhase string

const (
	PhaseIdle      CranePhase = "idle"
	PhaseTraveling CranePhase = "traveling"
	PhaseHoisting  CranePhase = "hoisting"
	PhaseForking   CranePhase = "forking"
	PhaseLoading   CranePhase = "loading"
	PhaseUnloading CranePhase = "unloading"
	PhaseDwell     CranePhase = "dwell"
	PhaseEmergency CranePhase = "emergency"
	PhaseFault     CranePhase = "fault"
)

type ForkPosition string

const (
	ForkRetracted ForkPosition = "retracted"
	ForkExtended  ForkPosition = "extended"
	ForkCentered  ForkPosition = "centered"
)

type Location struct {
	Aisle AisleID
	Bay   int
	Level int
	Depth int
}

func (l Location) String() string {
	return fmt.Sprintf("A%s B%d L%d D%d", l.Aisle, l.Bay, l.Level, l.Depth)
}

func (l Location) Valid() bool {
	return l.Aisle != "" && l.Bay > 0 && l.Level > 0 && l.Depth >= 0
}

func (l Location) Equal(other Location) bool {
	return l.Aisle == other.Aisle && l.Bay == other.Bay &&
		l.Level == other.Level && l.Depth == other.Depth
}

type CranePose struct {
	CraneID     CraneID
	Location    Location
	TravelMM    int64
	HoistMM     int64
	ForkMM      int64
	ForkPos     ForkPosition
	Loaded      bool
	PalletID    PalletID
	ProcessTick int64
	UpdatedAt   time.Time
}

func (p CranePose) Clone() CranePose { return p }

type MotionCommand struct {
	TaskID   TaskID
	CraneID  CraneID
	Axis     MotionAxis
	TargetMM int64
	SpeedMMS float64
}

type MotionProgress struct {
	Axis        MotionAxis
	StartMM     int64
	CurrentMM   int64
	TargetMM    int64
	StartedTick int64
	Complete    bool
}

func (m MotionProgress) RemainingMM() int64 {
	diff := m.TargetMM - m.CurrentMM
	if diff < 0 {
		return -diff
	}
	return diff
}

type StorageCell struct {
	Location   Location
	PalletID   PalletID
	Occupied   bool
	Reserved   bool
	ReservedBy CraneID
}

func (c StorageCell) Clone() StorageCell { return c }

type CraneStatus struct {
	CraneID     CraneID
	Phase       CranePhase
	Pose        CranePose
	Travel      MotionProgress
	Hoist       MotionProgress
	Fork        MotionProgress
	Interlocked bool
	EStopActive bool
	ActiveTask  TaskID
	Revision    uint64
	UpdatedAt   time.Time
}

func (s CraneStatus) Clone() CraneStatus {
	s.Pose = s.Pose.Clone()
	return s
}

type TransitionRequest struct {
	CraneID CraneID
	From    CranePhase
	To      CranePhase
	Tick    int64
	Force   bool
}

type TransitionResult struct {
	Accepted bool
	Phase    CranePhase
	Reason   string
	Revision uint64
}

type RetrievalTask struct {
	ID         TaskID
	CraneID    CraneID
	PalletID   PalletID
	Source     Location
	Dest       Location
	CreatedAt  time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time
}

type AlarmLevel int

const (
	AlarmInfo AlarmLevel = iota
	AlarmWarn
	AlarmCritical
)

type Alarm struct {
	CraneID CraneID
	Code    string
	Level   AlarmLevel
	Message string
	Since   time.Time
}

func (a Alarm) Clone() Alarm { return a }

type LimitSet struct {
	MaxTravelMM      int64
	MaxHoistMM       int64
	MaxForkMM        int64
	MaxTravelSpeed   float64
	MaxHoistSpeed    float64
	MaxForkSpeed     float64
	MinHoistClearMM  int64
	DwellWindowTicks int64
}

func DefaultLimits() LimitSet {
	return LimitSet{
		MaxTravelMM: 120000, MaxHoistMM: 18000, MaxForkMM: 2500,
		MaxTravelSpeed: 2000, MaxHoistSpeed: 800, MaxForkSpeed: 400,
		MinHoistClearMM: 500, DwellWindowTicks: 5,
	}
}

func (l LimitSet) Validate() error {
	if l.MaxTravelMM <= 0 {
		return errLimits("max_travel_mm")
	}
	if l.MaxHoistMM <= 0 {
		return errLimits("max_hoist_mm")
	}
	if l.MaxForkMM <= 0 {
		return errLimits("max_fork_mm")
	}
	if l.MaxTravelSpeed <= 0 || l.MaxHoistSpeed <= 0 || l.MaxForkSpeed <= 0 {
		return errLimits("speed")
	}
	if l.DwellWindowTicks < 0 {
		return errLimits("dwell_window_ticks")
	}
	return nil
}

type limitsError string

func (e limitsError) Error() string { return "model limits: " + string(e) }
func errLimits(field string) error  { return limitsError(field) }

func DeepCopyPoses(items []CranePose) []CranePose {
	if len(items) == 0 {
		return nil
	}
	out := make([]CranePose, len(items))
	copy(out, items)
	return out
}

func DeepCopyCells(items []StorageCell) []StorageCell {
	if len(items) == 0 {
		return nil
	}
	out := make([]StorageCell, len(items))
	copy(out, items)
	return out
}
