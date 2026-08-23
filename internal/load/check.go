package load

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lacsar712/stacklift/internal/model"
)

type Sensor struct {
	mu         sync.Mutex
	samples    map[string]model.LoadSample
	staleAfter time.Duration
}

func NewSensor() *Sensor {
	return &Sensor{samples: make(map[string]model.LoadSample), staleAfter: 2 * time.Second}
}

func (s *Sensor) Put(sample model.LoadSample) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.samples[sample.RigID] = sample.Clone()
}

func (s *Sensor) Get(rigID string) (model.LoadSample, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sample, ok := s.samples[rigID]
	if !ok {
		return model.LoadSample{}, false
	}
	return sample.Clone(), true
}

func (s *Sensor) Snapshot() map[string]model.LoadSample {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]model.LoadSample, len(s.samples))
	for id, sample := range s.samples {
		out[id] = sample.Clone()
	}
	return out
}

type Checker struct {
	sensor     *Sensor
	maxMoment  float64
	warnMoment float64
	staleAfter time.Duration
}

func NewChecker(sensor *Sensor, limits model.LimitSet) *Checker {
	return &Checker{sensor: sensor, maxMoment: limits.MaxMomentPct, warnMoment: limits.WarnMomentPct, staleAfter: 2 * time.Second}
}

func (c *Checker) ValidateSequence(ctx context.Context, rigID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	sample, ok := c.sensor.Get(rigID)
	if !ok || sample.Stale || time.Since(sample.At) > c.staleAfter {
		return Wrap(rigID, ErrStaleLoad)
	}
	if err := c.preflightMoment(ctx, rigID, sample); err != nil {
		return err
	}
	return c.confirmMoment(ctx, rigID, sample)
}

func (c *Checker) preflightMoment(ctx context.Context, rigID string, sample model.LoadSample) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if sample.MomentPct > c.maxMoment {
		return Wrap(rigID, ErrMomentExceeded)
	}
	return nil
}

func (c *Checker) confirmMoment(ctx context.Context, rigID string, sample model.LoadSample) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	sample2, ok := c.sensor.Get(rigID)
	if !ok {
		return Wrap(rigID, ErrStaleLoad)
	}
	if sample2.MomentPct > c.maxMoment {
		return Wrap(rigID, ErrMomentExceeded)
	}
	if sample2.MomentPct > c.warnMoment && sample.MomentPct > c.warnMoment {
		return fmt.Errorf("load %s: moment warning %.1f%%", rigID, sample2.MomentPct)
	}
	return nil
}

func (c *Checker) MomentOf(rigID string) (float64, error) {
	sample, ok := c.sensor.Get(rigID)
	if !ok {
		return 0, Wrap(rigID, ErrStaleLoad)
	}
	return sample.MomentPct, nil
}
