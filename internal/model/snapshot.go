package model

import "time"

type WarehouseSnapshot struct {
	Cranes      map[CraneID]CraneStatus
	Cells       []StorageCell
	Alarms      []Alarm
	Seq         uint64
	Captured    time.Time
	ProcessTick int64
}

func NewWarehouseSnapshot(seq uint64, tick int64) WarehouseSnapshot {
	return WarehouseSnapshot{
		Cranes: make(map[CraneID]CraneStatus), Seq: seq,
		Captured: time.Now(), ProcessTick: tick,
	}
}

func (s WarehouseSnapshot) Clone() WarehouseSnapshot {
	out := WarehouseSnapshot{
		Cranes: make(map[CraneID]CraneStatus, len(s.Cranes)),
		Cells:  DeepCopyCells(s.Cells), Alarms: make([]Alarm, len(s.Alarms)),
		Seq: s.Seq, Captured: s.Captured, ProcessTick: s.ProcessTick,
	}
	for id, st := range s.Cranes {
		out.Cranes[id] = st.Clone()
	}
	copy(out.Alarms, s.Alarms)
	return out
}

func (s WarehouseSnapshot) Crane(id CraneID) (CraneStatus, bool) {
	st, ok := s.Cranes[id]
	if !ok {
		return CraneStatus{}, false
	}
	return st.Clone(), true
}

func (s WarehouseSnapshot) CellAt(loc Location) (StorageCell, bool) {
	for _, c := range s.Cells {
		if c.Location.Equal(loc) {
			return c.Clone(), true
		}
	}
	return StorageCell{}, false
}

type CraneSnapshot struct {
	Status CraneStatus
	Seq    uint64
	Tick   int64
}

func (s CraneSnapshot) Clone() CraneSnapshot {
	s.Status = s.Status.Clone()
	return s
}

type MotionPlan struct {
	TaskID      TaskID
	CraneID     CraneID
	From        Position
	To          Position
	Steps       []MotionStep
	EstimatedMS int64
}

func (p MotionPlan) Clone() MotionPlan {
	out := MotionPlan{TaskID: p.TaskID, CraneID: p.CraneID, From: p.From, To: p.To, EstimatedMS: p.EstimatedMS}
	if len(p.Steps) > 0 {
		out.Steps = make([]MotionStep, len(p.Steps))
		copy(out.Steps, p.Steps)
	}
	return out
}

type MotionStep struct {
	Axis     MotionAxis
	FromMM   int64
	ToMM     int64
	SpeedMMS float64
	Order    int
}

func (s MotionStep) DurationMS() int64 {
	delta := s.ToMM - s.FromMM
	if delta < 0 {
		delta = -delta
	}
	if s.SpeedMMS <= 0 {
		return 0
	}
	return int64(float64(delta) / s.SpeedMMS * 1000)
}
