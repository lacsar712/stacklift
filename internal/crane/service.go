// Package crane manages tower crane rig services.
package crane

import (
	"context"
	"sync"
	"time"

	"github.com/lacsar712/stacklift/internal/boom"
	"github.com/lacsar712/stacklift/internal/fsm"
	"github.com/lacsar712/stacklift/internal/hook"
	"github.com/lacsar712/stacklift/internal/load"
	"github.com/lacsar712/stacklift/internal/model"
	"github.com/lacsar712/stacklift/internal/slew"
	"github.com/lacsar712/stacklift/internal/store"
	"github.com/lacsar712/stacklift/internal/wind"
)

type Status struct {
	RigID, Mode     string
	Azimuth, RadiusM, MomentPct, WindMS float64
	Revision          uint64
	duty              model.DutyMode
}

type Rig struct {
	ID      string
	Emitter *slew.Emitter
	Boom    *boom.Driver
	FSM     *fsm.DutyMachine
	Hook    *hook.Sensor
	Anemo   *wind.Anemometer
	Store   *store.RigStore
}

type Group struct {
	mu    sync.Mutex
	rigs  map[string]*Rig
	order []string
}

func NewGroup() *Group { return &Group{rigs: make(map[string]*Rig)} }

func (g *Group) Register(rig *Rig) {
	g.mu.Lock()
	g.rigs[rig.ID] = rig
	g.order = append(g.order, rig.ID)
	g.mu.Unlock()
}

func (g *Group) Get(id string) (*Rig, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	r, ok := g.rigs[id]
	return r, ok
}

func (g *Group) IDs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]string, len(g.order))
	copy(out, g.order)
	return out
}

func (g *Group) StatusAll() []Status {
	g.mu.Lock()
	ids := append([]string(nil), g.order...)
	g.mu.Unlock()
	out := make([]Status, 0, len(ids))
	for _, id := range ids {
		rig, ok := g.Get(id)
		if !ok {
			continue
		}
		st := Status{RigID: id, Azimuth: rig.Emitter.AngleOf(id)}
		mode, rev, _ := rig.FSM.Snapshot(id)
		st.duty = mode
		st.Mode = string(mode)
		st.Revision = rev
		if pose, ok := rig.Boom.PoseOf(id); ok {
			st.RadiusM = pose.RadiusM
		}
		if p, ok := rig.Hook.Last(id); ok {
			st.MomentPct = p.MomentPct
		}
		if w, ok := rig.Anemo.Last(id); ok {
			st.WindMS = w.SpeedMS
		}
		out = append(out, st)
	}
	return out
}

type Service struct {
	group       *Group
	coordinator *slew.Coordinator
	loadSensor  *load.Sensor
	fsm         *fsm.DutyMachine
	store       *store.RigStore
	activeCtx   context.Context
	cancel      context.CancelFunc
	mu          sync.Mutex
}

func NewService(group *Group, coord *slew.Coordinator, sensor *load.Sensor, machine *fsm.DutyMachine, store *store.RigStore) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{group: group, coordinator: coord, loadSensor: sensor, fsm: machine, store: store, activeCtx: ctx, cancel: cancel}
}

func (s *Service) RequestSlew(ctx context.Context, rigID string, targetAz, targetRadius float64, reason string) error {
	parent := s.activeCtx
	if ctx != nil {
		parent = ctx
	}
	if err := parent.Err(); err != nil {
		return err
	}
	req := slew.RunRequest{RigID: rigID, TargetAzDeg: targetAz, TargetRadiusM: targetRadius, Reason: reason, Luff: targetRadius > 0}
	if err := s.coordinator.Run(parent, req); err != nil {
		if load.IsMoment(err) {
			return load.Wrap(rigID, load.ErrMomentExceeded)
		}
		return err
	}
	s.syncPose(rigID)
	return nil
}

func (s *Service) EmergencyStop(rigID string, tick int64) error {
	s.mu.Lock()
	s.cancel()
	s.activeCtx, s.cancel = context.WithCancel(context.Background())
	s.mu.Unlock()
	_, err := s.fsm.Transition(s.activeCtx, model.TransitionRequest{RigID: rigID, To: model.DutyEmergency, Tick: tick, Force: true})
	return err
}

func (s *Service) Transition(ctx context.Context, req model.TransitionRequest) (model.TransitionResult, error) {
	return s.fsm.Transition(ctx, req)
}

func (s *Service) IngestLoad(sample model.LoadSample) { s.loadSensor.Put(sample) }

func (s *Service) syncPose(rigID string) {
	rig, ok := s.group.Get(rigID)
	if !ok {
		return
	}
	pose, _ := rig.Boom.PoseOf(rigID)
	moment := 0.0
	if p, ok := rig.Hook.Last(rigID); ok {
		moment = p.MomentPct
	}
	s.store.PutPose(model.RigPose{
		RigID: rigID, AzimuthDeg: rig.Emitter.AngleOf(rigID), RadiusM: pose.RadiusM,
		HookHeightM: pose.HookHeightM, BoomAngleDeg: pose.BoomAngleDeg, MomentPct: moment, UpdatedAt: time.Now().UTC(),
	})
}

func (s *Service) ActiveContext() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeCtx
}

func (s *Service) Bootstrap() {
	for _, id := range s.group.IDs() {
		rig, ok := s.group.Get(id)
		if !ok {
			continue
		}
		rig.Emitter.SetAngle(id, 0)
		rig.Boom.Ensure(id)
		rig.FSM.Ensure(id)
		s.syncPose(id)
	}
}

type Builder struct {
	towerH, ratedKg, rateDeg float64
	limits                   model.LimitSet
	windHold                 int64
}

func NewBuilder(towerH, ratedKg, rateDeg float64, limits model.LimitSet, windHold int64) *Builder {
	return &Builder{towerH, ratedKg, rateDeg, limits, windHold}
}

func (b *Builder) Build(id string, emitter *slew.Emitter, machine *fsm.DutyMachine, st *store.RigStore) *Rig {
	geom := boom.NewGeometry(b.limits.MinRadiusM, b.limits.MaxRadiusM, b.limits.MinBoomAngleDeg, b.limits.MaxBoomAngleDeg)
	driver := boom.NewDriver(geom, b.towerH, b.rateDeg)
	anemo := wind.NewAnemometer(b.limits.MaxWindMS, b.limits.GustFactor, b.windHold)
	hookSensor := hook.NewSensor(b.ratedKg)
	rig := &Rig{ID: id, Emitter: emitter, Boom: driver, FSM: machine, Hook: hookSensor, Anemo: anemo, Store: st}
	emitter.SetAngle(id, 0)
	driver.Ensure(id)
	machine.Ensure(id)
	return rig
}

type SiteSummary struct {
	RigCount               int
	Emergency, WindHold    bool
	MaxMoment, MaxWindMS   float64
}

func Summarize(group *Group) SiteSummary {
	s := SiteSummary{RigCount: len(group.IDs())}
	for _, st := range group.StatusAll() {
		if st.duty == model.DutyEmergency {
			s.Emergency = true
		}
		if st.duty == model.DutyWindHold {
			s.WindHold = true
		}
		if st.MomentPct > s.MaxMoment {
			s.MaxMoment = st.MomentPct
		}
		if st.WindMS > s.MaxWindMS {
			s.MaxWindMS = st.WindMS
		}
	}
	return s
}
