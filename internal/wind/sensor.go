package wind

import (
	"sync"
	"time"

	"github.com/lacsar712/stacklift/internal/model"
)

type Anemometer struct {
	mu         sync.Mutex
	maxMS      float64
	gustFactor float64
	holdTicks  int64
	last       map[string]model.WindSample
	gustStart  map[string]int64
	sustained  map[string]int64
	staleAfter time.Duration
}

func NewAnemometer(maxMS, gustFactor float64, holdTicks int64) *Anemometer {
	return &Anemometer{
		maxMS: maxMS, gustFactor: gustFactor, holdTicks: holdTicks,
		last: make(map[string]model.WindSample), gustStart: make(map[string]int64),
		sustained: make(map[string]int64), staleAfter: 3 * time.Second,
	}
}

func (a *Anemometer) Ingest(sample model.WindSample) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.last[sample.RigID] = sample.Clone()
	speed, gust := sample.SpeedMS, sample.GustMS
	if gust <= 0 {
		gust = speed
	}
	if speed > a.maxMS {
		a.sustained[sample.RigID] = sample.ProcessTick
		return Wrap(sample.RigID, ErrSustainedHigh)
	}
	gustLimit := a.maxMS * a.gustFactor
	if gust > gustLimit {
		if _, ok := a.gustStart[sample.RigID]; !ok {
			a.gustStart[sample.RigID] = sample.ProcessTick
		}
		if sample.ProcessTick-a.gustStart[sample.RigID] >= a.holdTicks {
			a.sustained[sample.RigID] = sample.ProcessTick
			return Wrap(sample.RigID, ErrSustainedHigh)
		}
		return Wrap(sample.RigID, ErrWindGust)
	}
	delete(a.gustStart, sample.RigID)
	if tick, ok := a.sustained[sample.RigID]; ok && sample.ProcessTick-tick < a.holdTicks {
		return Wrap(sample.RigID, ErrSustainedHigh)
	}
	delete(a.sustained, sample.RigID)
	return nil
}

func (a *Anemometer) Last(rigID string) (model.WindSample, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	s, ok := a.last[rigID]
	if !ok {
		return model.WindSample{}, false
	}
	return s.Clone(), true
}

func (a *Anemometer) Snapshot() map[string]model.WindSample {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make(map[string]model.WindSample, len(a.last))
	for id, s := range a.last {
		out[id] = s.Clone()
	}
	return out
}

func (a *Anemometer) GustActive(rigID string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, ok := a.gustStart[rigID]
	return ok
}

func (a *Anemometer) SetStaleAfter(d time.Duration) {
	a.mu.Lock()
	a.staleAfter = d
	a.mu.Unlock()
}
