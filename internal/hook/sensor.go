// Package hook models hook payload sensing.
package hook

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/lacsar712/stacklift/internal/model"
)

type Payload struct {
	RigID       string
	MassKg      float64
	MomentPct   float64
	RadiusM     float64
	HeightM     float64
	At          time.Time
	ProcessTick int64
}

func (p Payload) Clone() Payload { return p }

type Sensor struct {
	mu       sync.Mutex
	readings map[string]Payload
	ratedKg  float64
}

func NewSensor(ratedLoadKg float64) *Sensor {
	return &Sensor{readings: make(map[string]Payload), ratedKg: ratedLoadKg}
}

func (s *Sensor) Ingest(rigID string, massKg, radiusM, heightM float64, tick int64) Payload {
	s.mu.Lock()
	defer s.mu.Unlock()
	moment := 0.0
	if s.ratedKg > 0 {
		moment = (massKg / s.ratedKg) * 100
		if radiusM > 0 {
			moment *= radiusM / 30
		}
	}
	p := Payload{RigID: rigID, MassKg: massKg, MomentPct: moment, RadiusM: radiusM, HeightM: heightM, At: time.Now().UTC(), ProcessTick: tick}
	s.readings[rigID] = p
	return p.Clone()
}

func (s *Sensor) Last(rigID string) (Payload, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.readings[rigID]
	if !ok {
		return Payload{}, false
	}
	return p.Clone(), true
}

func (s *Sensor) ToLoadSample(rigID string, stale bool) (model.LoadSample, bool) {
	p, ok := s.Last(rigID)
	if !ok {
		return model.LoadSample{}, false
	}
	return model.LoadSample{RigID: rigID, MassKg: p.MassKg, MomentPct: p.MomentPct, Stale: stale, ProcessTick: p.ProcessTick, At: p.At}, true
}

type Estimator struct {
	towerHeightM, boomLengthM, ratedKg float64
}

func NewEstimator(towerH, boomLen, ratedKg float64) *Estimator {
	return &Estimator{towerH, boomLen, ratedKg}
}

func (e *Estimator) MomentPercent(massKg, radiusM float64) float64 {
	if e.ratedKg <= 0 {
		return 0
	}
	return (massKg / e.ratedKg) * 100 * (radiusM / (e.boomLengthM * 0.8))
}

func (e *Estimator) ValidateMass(massKg float64) error {
	if massKg < 0 {
		return fmt.Errorf("hook: negative mass")
	}
	if massKg > e.ratedKg*1.1 {
		return fmt.Errorf("hook: mass exceeds rating")
	}
	return nil
}

func (e *Estimator) HookHeight(boomAngleDeg, dropM float64) float64 {
	rad := boomAngleDeg * math.Pi / 180
	return e.towerHeightM - dropM - e.boomLengthM*math.Sin(rad)
}
