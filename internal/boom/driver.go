// Package boom models tower crane luff (boom) motion.
package boom

import (
	"context"
	"fmt"
	"math"
	"sync"
)

type Pose struct {
	RadiusM      float64
	BoomAngleDeg float64
	HookHeightM  float64
}

func (p Pose) Clone() Pose { return p }

type Geometry struct {
	limits MinMax
}

type MinMax struct {
	MinRadiusM, MaxRadiusM, MinBoomAngleDeg, MaxBoomAngleDeg float64
}

func NewGeometry(minR, maxR, minA, maxA float64) *Geometry {
	return &Geometry{limits: MinMax{minR, maxR, minA, maxA}}
}

func (g *Geometry) RadiusFromAngle(angleDeg, towerHeightM float64) float64 {
	rad := angleDeg * math.Pi / 180.0
	if rad <= 0 {
		return g.limits.MinRadiusM
	}
	r := towerHeightM / math.Tan(rad)
	if r < g.limits.MinRadiusM {
		return g.limits.MinRadiusM
	}
	if r > g.limits.MaxRadiusM {
		return g.limits.MaxRadiusM
	}
	return r
}

func (g *Geometry) AngleFromRadius(radiusM, towerHeightM float64) float64 {
	if radiusM <= 0 {
		return g.limits.MaxBoomAngleDeg
	}
	rad := math.Atan2(towerHeightM, radiusM)
	deg := rad * 180.0 / math.Pi
	if deg < g.limits.MinBoomAngleDeg {
		return g.limits.MinBoomAngleDeg
	}
	if deg > g.limits.MaxBoomAngleDeg {
		return g.limits.MaxBoomAngleDeg
	}
	return deg
}

func (g *Geometry) HookHeight(towerHeightM, boomAngleDeg, dropM float64) float64 {
	rad := boomAngleDeg * math.Pi / 180.0
	if math.Sin(rad) == 0 {
		return towerHeightM - dropM
	}
	boomLen := towerHeightM / math.Sin(rad)
	return towerHeightM - dropM - boomLen*math.Sin(rad)
}

func (g *Geometry) ClampRadius(r float64) float64 {
	if r < g.limits.MinRadiusM {
		return g.limits.MinRadiusM
	}
	if r > g.limits.MaxRadiusM {
		return g.limits.MaxRadiusM
	}
	return r
}

type MoveRequest struct {
	RigID, Reason     string
	TargetRadiusM     float64
	RateDegPerS       float64
}

type MoveResult struct {
	RigID       string
	FromRadiusM float64
	ToRadiusM   float64
	Steps       int
}

type Driver struct {
	mu      sync.Mutex
	poses   map[string]Pose
	geom    *Geometry
	towerH  float64
	rateDeg float64
	stepDeg float64
	log     []MoveEvent
	active  map[string]context.CancelFunc
}

type MoveEvent struct {
	RigID, Kind, Detail string
}

func NewDriver(geom *Geometry, towerHeightM, rateDegPerS float64) *Driver {
	return &Driver{
		poses: make(map[string]Pose), geom: geom, towerH: towerHeightM,
		rateDeg: rateDegPerS, stepDeg: 0.5, log: []MoveEvent{}, active: make(map[string]context.CancelFunc),
	}
}

func (d *Driver) Ensure(rigID string) Pose {
	d.mu.Lock()
	defer d.mu.Unlock()
	if p, ok := d.poses[rigID]; ok {
		return p.Clone()
	}
	p := Pose{RadiusM: d.geom.limits.MaxRadiusM * 0.6, BoomAngleDeg: 45, HookHeightM: d.geom.HookHeight(d.towerH, 45, 0)}
	d.poses[rigID] = p
	return p.Clone()
}

func (d *Driver) PoseOf(rigID string) (Pose, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	p, ok := d.poses[rigID]
	if !ok {
		return Pose{}, false
	}
	return p.Clone(), true
}

func (d *Driver) Move(ctx context.Context, req MoveRequest) (MoveResult, error) {
	d.mu.Lock()
	cur, ok := d.poses[req.RigID]
	if !ok {
		cur = d.Ensure(req.RigID)
	}
	target := d.geom.ClampRadius(req.TargetRadiusM)
	rate := req.RateDegPerS
	if rate <= 0 {
		rate = d.rateDeg
	}
	runCtx, cancel := context.WithCancel(ctx)
	if old, ok := d.active[req.RigID]; ok {
		old()
	}
	d.active[req.RigID] = cancel
	d.log = append(d.log, MoveEvent{req.RigID, "start", fmt.Sprintf("to=%.2f", target)})
	d.mu.Unlock()
	defer func() {
		d.mu.Lock()
		delete(d.active, req.RigID)
		d.mu.Unlock()
		cancel()
	}()
	from, steps := cur.RadiusM, 0
	stepM := 0.5
	for math.Abs(target-cur.RadiusM) > 0.05 {
		select {
		case <-runCtx.Done():
			return MoveResult{RigID: req.RigID, FromRadiusM: from, ToRadiusM: cur.RadiusM, Steps: steps}, fmt.Errorf("boom cancelled: %w", runCtx.Err())
		default:
		}
		delta := target - cur.RadiusM
		rStep := stepM
		if delta < 0 {
			rStep = -rStep
		}
		if math.Abs(delta) <= stepM {
			cur.RadiusM = target
		} else {
			cur.RadiusM += rStep
		}
		angle := d.geom.AngleFromRadius(cur.RadiusM, d.towerH)
		cur.BoomAngleDeg = angle
		cur.HookHeightM = d.geom.HookHeight(d.towerH, angle, 0)
		steps++
		d.mu.Lock()
		d.poses[req.RigID] = cur
		d.mu.Unlock()
		if steps > 500 {
			return MoveResult{RigID: req.RigID, FromRadiusM: from, ToRadiusM: cur.RadiusM, Steps: steps}, fmt.Errorf("boom: move did not converge")
		}
	}
	return MoveResult{RigID: req.RigID, FromRadiusM: from, ToRadiusM: cur.RadiusM, Steps: steps}, nil
}

func (d *Driver) Stop(rigID string) {
	d.mu.Lock()
	cancel, ok := d.active[rigID]
	d.mu.Unlock()
	if ok {
		cancel()
	}
}

func (d *Driver) Events() []MoveEvent {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]MoveEvent, len(d.log))
	copy(out, d.log)
	return out
}
