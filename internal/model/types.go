// Package model defines shared crane rig domain types.
package model

import "time"

type DutyMode string

const (
	DutyIdle      DutyMode = "idle"
	DutyHoist     DutyMode = "hoist"
	DutySlew      DutyMode = "slew"
	DutyLuff      DutyMode = "luff"
	DutyEmergency DutyMode = "emergency"
	DutyWindHold  DutyMode = "wind_hold"
	DutyFault     DutyMode = "fault"
)

type RigPose struct {
	RigID        string
	AzimuthDeg   float64
	RadiusM      float64
	HookHeightM  float64
	BoomAngleDeg float64
	MomentPct    float64
	ProcessTick  int64
	UpdatedAt    time.Time
}

func (p RigPose) Clone() RigPose { return p }

type LoadSample struct {
	RigID       string
	MassKg      float64
	MomentPct   float64
	Stale       bool
	ProcessTick int64
	At          time.Time
}

func (s LoadSample) Clone() LoadSample { return s }

type WindSample struct {
	RigID        string
	SpeedMS      float64
	GustMS       float64
	DirectionDeg float64
	ProcessTick  int64
	At           time.Time
}

func (s WindSample) Clone() WindSample { return s }

type TransitionRequest struct {
	RigID string
	From  DutyMode
	To    DutyMode
	Tick  int64
	Force bool
}

type TransitionResult struct {
	Accepted bool
	Mode     DutyMode
	Reason   string
	Revision uint64
}

type AlarmLevel int

const (
	AlarmInfo AlarmLevel = iota
	AlarmWarn
	AlarmCritical
)

type Alarm struct {
	RigID   string
	Code    string
	Level   AlarmLevel
	Message string
	Since   time.Time
}

func (a Alarm) Clone() Alarm { return a }

type RigStatus struct {
	RigID       string
	Mode        DutyMode
	Pose        RigPose
	Load        LoadSample
	Wind        WindSample
	MomentPct   float64
	WindHold    bool
	Interlocked bool
	Revision    uint64
}

func (s RigStatus) Clone() RigStatus {
	s.Pose = s.Pose.Clone()
	s.Load = s.Load.Clone()
	s.Wind = s.Wind.Clone()
	return s
}

type LimitSet struct {
	MaxMomentPct, WarnMomentPct, MaxWindMS, GustFactor float64
	MaxRadiusM, MinRadiusM, MaxBoomAngleDeg, MinBoomAngleDeg float64
}

func DefaultLimits() LimitSet {
	return LimitSet{100, 85, 13.8, 1.35, 60, 5, 75, 15}
}

func (l LimitSet) Validate() error {
	if l.MaxMomentPct <= 0 {
		return errLimits("max_moment_pct")
	}
	if l.WarnMomentPct <= 0 || l.WarnMomentPct > l.MaxMomentPct {
		return errLimits("warn_moment_pct")
	}
	if l.MaxWindMS <= 0 {
		return errLimits("max_wind_ms")
	}
	if l.GustFactor < 1 {
		return errLimits("gust_factor")
	}
	if l.MaxRadiusM <= l.MinRadiusM {
		return errLimits("radius")
	}
	return nil
}

type limitsError string

func (e limitsError) Error() string { return "model limits: " + string(e) }
func errLimits(msg string) error    { return limitsError(msg) }

func DeepCopyPose(items []RigPose) []RigPose {
	if len(items) == 0 {
		return nil
	}
	out := make([]RigPose, len(items))
	copy(out, items)
	return out
}
