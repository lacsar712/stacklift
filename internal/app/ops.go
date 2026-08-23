package app

import (
	"context"
	"errors"

	"github.com/lacsar712/stacklift/internal/counter"
	"github.com/lacsar712/stacklift/internal/load"
	"github.com/lacsar712/stacklift/internal/model"
	"github.com/lacsar712/stacklift/internal/wind"
)

func (s *Service) HandleFault(rigID string, err error) string {
	if err == nil {
		return "ok"
	}
	switch {
	case errors.Is(err, load.ErrMomentExceeded):
		return "moment"
	case errors.Is(err, load.ErrStaleLoad):
		return "stale_load"
	case errors.Is(err, wind.ErrWindGust):
		return "wind_gust"
	case errors.Is(err, wind.ErrSustainedHigh):
		return "wind_ban"
	case errors.Is(err, counter.ErrCycleLimit):
		return "cycle_limit"
	default:
		return "unknown"
	}
}

func (s *Service) WindRecoverable(err error) bool { return wind.IsRecoverable(err) }
func (s *Service) MomentExceeded(err error) bool  { return load.IsMoment(err) }
func (s *Service) StaleLoad(err error) bool     { return load.IsStale(err) }
func (s *Service) WindHoldClosed(rigID string) bool { return s.WindWindow.Closed(rigID) }
func (s *Service) ProcessTick() int64             { return s.Clock.Tick() }
func (s *Service) AdvanceClock(n int64) int64     { return s.Clock.Advance(n) }
func (s *Service) RootContext() context.Context   { return s.rootCtx }

func (s *Service) CycleOnce(samples map[string]model.LoadSample, winds map[string]model.WindSample) {
	for _, sample := range samples {
		s.IngestLoad(sample)
	}
	for _, w := range winds {
		_ = s.IngestWind(w)
	}
	s.TickOnce()
}
