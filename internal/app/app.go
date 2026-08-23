// Package app wires stacklift subsystems into a runnable site service.
package app

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/lacsar712/stacklift/internal/alarm"
	"github.com/lacsar712/stacklift/internal/api"
	"github.com/lacsar712/stacklift/internal/boom"
	"github.com/lacsar712/stacklift/internal/clock"
	"github.com/lacsar712/stacklift/internal/config"
	"github.com/lacsar712/stacklift/internal/counter"
	"github.com/lacsar712/stacklift/internal/crane"
	"github.com/lacsar712/stacklift/internal/fsm"
	"github.com/lacsar712/stacklift/internal/hook"
	"github.com/lacsar712/stacklift/internal/interlock"
	"github.com/lacsar712/stacklift/internal/journal"
	"github.com/lacsar712/stacklift/internal/load"
	"github.com/lacsar712/stacklift/internal/model"
	"github.com/lacsar712/stacklift/internal/slew"
	"github.com/lacsar712/stacklift/internal/store"
	"github.com/lacsar712/stacklift/internal/telemetry"
	"github.com/lacsar712/stacklift/internal/wind"
)

type Service struct {
	Cfg         config.Config
	Clock       *clock.ProcessClock
	Store       *store.RigStore
	Group       *crane.Group
	CraneSvc    *crane.Service
	Coordinator *slew.Coordinator
	FSM         *fsm.DutyMachine
	Emitter     *slew.Emitter
	Plant       *slew.Plant
	BoomDriver  *boom.Driver
	Interlock   *interlock.Guard
	WindWindow  *interlock.WindWindow
	LoadSensor  *load.Sensor
	Checker     *load.Checker
	Anemo       map[string]*wind.Anemometer
	HookSensor  map[string]*hook.Sensor
	Counter     *counter.CycleCounter
	Alarms      *alarm.Bus
	Telemetry   *telemetry.Bus
	Journal     *journal.Writer
	Server      *api.Server
	HTTP        *http.Server
	rootCtx     context.Context
	cancel      context.CancelFunc
	mu          sync.Mutex
	started     bool
}

func New(cfg config.Config) (*Service, error) {
	cfg = config.FromEnv(cfg)
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	root, cancel := context.WithCancel(context.Background())
	clk := clock.New(cfg.ProcessTickMS)
	st := store.NewRigStore()
	emitter := slew.NewEmitter()
	plant := slew.NewPlant(emitter, 5)
	limits := interlock.LimitsFromConfig(cfg)
	windWindow := interlock.NewWindWindow(limits, clk)
	guard := interlock.NewGuard(limits, windWindow)
	loadSensor := load.NewSensor()
	checker := load.NewChecker(loadSensor, cfg.Limits)
	geom := boom.NewGeometry(cfg.Limits.MinRadiusM, cfg.Limits.MaxRadiusM, cfg.Limits.MinBoomAngleDeg, cfg.Limits.MaxBoomAngleDeg)
	boomDriver := boom.NewDriver(geom, 40, 5)
	coord := slew.NewCoordinator(plant, boomDriver, guard, checker, 5)
	machine := fsm.NewDutyMachine(emitter)
	group := crane.NewGroup()
	builder := crane.NewBuilder(40, 12000, 5, cfg.Limits, cfg.WindHoldWindow)
	anemoMap := make(map[string]*wind.Anemometer)
	hookMap := make(map[string]*hook.Sensor)
	for _, id := range cfg.RigIDs {
		rig := builder.Build(id, emitter, machine, st)
		group.Register(rig)
		anemoMap[id] = rig.Anemo
		hookMap[id] = rig.Hook
		guard.SetMode(id, model.DutyIdle)
	}
	craneSvc := crane.NewService(group, coord, loadSensor, machine, st)
	alarms := alarm.NewBus(30 * time.Second)
	telem := telemetry.NewBus(cfg.TelemetryBuffer)
	journalW := journal.NewWriter(cfg.JournalPath, 2000)
	s := &Service{
		Cfg: cfg, Clock: clk, Store: st, Group: group, CraneSvc: craneSvc,
		Coordinator: coord, FSM: machine, Emitter: emitter, Plant: plant, BoomDriver: boomDriver,
		Interlock: guard, WindWindow: windWindow, LoadSensor: loadSensor, Checker: checker,
		Anemo: anemoMap, HookSensor: hookMap, Counter: counter.New(500),
		Alarms: alarms, Telemetry: telem, Journal: journalW, rootCtx: root, cancel: cancel,
	}
	s.Server = api.New(cfg, s, group, st, guard, alarms, telem, journalW, clk)
	craneSvc.Bootstrap()
	return s, nil
}

func (s *Service) RequestSlew(ctx context.Context, rigID string, targetAz, targetRadius float64, reason string) error {
	if !s.Cfg.RigKnown(rigID) {
		return fmt.Errorf("app: unknown rig %s", rigID)
	}
	if err := s.CraneSvc.RequestSlew(ctx, rigID, targetAz, targetRadius, reason); err != nil {
		return err
	}
	tick := s.Clock.Tick()
	s.Telemetry.Emit("app", "slew", rigID, tick, reason)
	s.Journal.Append(tick, rigID, "slew", reason)
	return nil
}

func (s *Service) Transition(ctx context.Context, rigID string, to model.DutyMode) (model.TransitionResult, error) {
	from := s.FSM.ModeOf(rigID)
	res, err := s.FSM.Transition(ctx, model.TransitionRequest{RigID: rigID, From: from, To: to, Tick: s.Clock.Tick()})
	if res.Accepted {
		s.Interlock.SetMode(rigID, to)
		s.Journal.Append(s.Clock.Tick(), rigID, "transition", string(to))
	}
	return res, err
}

func (s *Service) EmergencyStop(rigID string) error {
	tick := s.Clock.Tick()
	if err := s.CraneSvc.EmergencyStop(rigID, tick); err != nil {
		return err
	}
	s.Interlock.SetMode(rigID, model.DutyEmergency)
	s.Alarms.RaiseEmergency(rigID)
	s.Telemetry.Emit("app", "emergency", rigID, tick, "stop")
	return nil
}

func (s *Service) IngestWind(sample model.WindSample) error {
	anemo, ok := s.Anemo[sample.RigID]
	if !ok {
		return fmt.Errorf("app: unknown rig %s", sample.RigID)
	}
	err := anemo.Ingest(sample)
	s.Interlock.SetWindFault(sample.RigID, err)
	if err != nil {
		s.Alarms.RaiseWind(sample.RigID, wind.Classify(err), sample.SpeedMS)
		if wind.IsBan(err) {
			_, _ = s.Transition(s.rootCtx, sample.RigID, model.DutyWindHold)
		}
	}
	return err
}

func (s *Service) IngestLoad(sample model.LoadSample) {
	s.LoadSensor.Put(sample)
	s.CraneSvc.IngestLoad(sample)
	if sample.MomentPct > s.Cfg.Limits.MaxMomentPct {
		s.Interlock.SetLoadFault(sample.RigID, load.Wrap(sample.RigID, load.ErrMomentExceeded))
		s.Alarms.RaiseMoment(sample.RigID, sample.MomentPct)
	} else if sample.Stale {
		s.Interlock.SetLoadFault(sample.RigID, load.Wrap(sample.RigID, load.ErrStaleLoad))
		s.Alarms.RaiseStaleLoad(sample.RigID)
	} else {
		s.Interlock.SetLoadFault(sample.RigID, nil)
	}
}

func (s *Service) TickOnce() {
	tick := s.Clock.AdvanceOne()
	s.Alarms.Tick(time.Now())
	for _, id := range s.Cfg.RigIDs {
		if hookS, ok := s.HookSensor[id]; ok {
			if p, ok := hookS.Last(id); ok {
				sample, _ := hookS.ToLoadSample(id, false)
				sample.MomentPct = p.MomentPct
				s.IngestLoad(sample)
			}
		}
		if anemo, ok := s.Anemo[id]; ok {
			if w, ok := anemo.Last(id); ok {
				_ = s.IngestWind(w)
			}
		}
		st := model.RigStatus{RigID: id, Mode: s.FSM.ModeOf(id)}
		if pose, ok := s.Store.GetPose(id); ok {
			st.Pose = pose
			st.MomentPct = pose.MomentPct
		}
		if sample, ok := s.LoadSensor.Get(id); ok {
			st.Load = sample
		}
		st.WindHold = s.WindWindow.Active(id)
		st.Interlocked = len(s.Interlock.Eval(id, interlock.MotionSlew)) > 0
		s.Store.PutStatus(st)
	}
	_ = tick
}

func (s *Service) Cancel() { s.cancel() }
