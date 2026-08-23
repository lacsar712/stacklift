package slew

import (
	"context"
	"fmt"
	"sync"
)

type Plan struct {
	RigID, Reason       string
	FromDeg, ToDeg, RateDeg float64
	Steps               []float64
}

func NewPlan(rigID string, from, to, rate float64, reason string) Plan {
	return Plan{RigID: rigID, FromDeg: from, ToDeg: to, RateDeg: rate, Reason: reason, Steps: buildSteps(from, to, rate)}
}

func buildSteps(from, to, rate float64) []float64 {
	if rate <= 0 {
		rate = 5
	}
	delta := normalizeDelta(to - from)
	if delta == 0 {
		return nil
	}
	step := rate * 0.1
	if step <= 0 {
		step = 1
	}
	var steps []float64
	cur, rem := from, delta
	for rem > 0 {
		s := step
		if rem < s {
			s = rem
		}
		if delta < 0 {
			s = -s
		}
		cur += s
		steps = append(steps, normalizeDeg(cur))
		rem -= step
	}
	if len(steps) == 0 || steps[len(steps)-1] != normalizeDeg(to) {
		steps = append(steps, normalizeDeg(to))
	}
	return steps
}

func normalizeDeg(d float64) float64 {
	for d < 0 {
		d += 360
	}
	for d >= 360 {
		d -= 360
	}
	return d
}

func normalizeDelta(d float64) float64 {
	for d > 180 {
		d -= 360
	}
	for d < -180 {
		d += 360
	}
	return d
}

type Emitter struct {
	mu     sync.Mutex
	angles map[string]float64
	log    []EmitEvent
	hooks  []func(context.Context, string, float64)
}

type EmitEvent struct {
	RigID, Kind string
	Azimuth     float64
}

func NewEmitter() *Emitter {
	return &Emitter{angles: make(map[string]float64), log: []EmitEvent{}}
}

func (e *Emitter) SetAngle(rigID string, deg float64) {
	e.mu.Lock()
	e.angles[rigID] = normalizeDeg(deg)
	e.mu.Unlock()
}

func (e *Emitter) AngleOf(rigID string) float64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.angles[rigID]
}

func (e *Emitter) OnEmit(fn func(context.Context, string, float64)) {
	e.mu.Lock()
	e.hooks = append(e.hooks, fn)
	e.mu.Unlock()
}

func (e *Emitter) Emit(ctx context.Context, rigID string, azimuthDeg float64) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("slew emit cancelled: %w", ctx.Err())
	default:
	}
	deg := normalizeDeg(azimuthDeg)
	e.mu.Lock()
	e.angles[rigID] = deg
	hooks := append([]func(context.Context, string, float64){}, e.hooks...)
	e.log = append(e.log, EmitEvent{rigID, "emit", deg})
	e.mu.Unlock()
	for _, fn := range hooks {
		fn(ctx, rigID, deg)
	}
	return nil
}

type Plant struct {
	mu      sync.Mutex
	emitter *Emitter
	log     []PlantEvent
}

type PlantEvent struct {
	RigID, Kind, Detail string
}

func NewPlant(emitter *Emitter, _ float64) *Plant {
	return &Plant{emitter: emitter, log: []PlantEvent{}}
}

func (p *Plant) Run(ctx context.Context, plan Plan) error {
	for i, step := range plan.Steps {
		select {
		case <-ctx.Done():
			return fmt.Errorf("slew plant cancelled at step %d: %w", i, ctx.Err())
		default:
		}
		if err := p.emitter.Emit(ctx, plan.RigID, step); err != nil {
			return err
		}
	}
	return nil
}
