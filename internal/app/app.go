package app

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/lacsar712/stacklift/internal/aisle"
	"github.com/lacsar712/stacklift/internal/clock"
	"github.com/lacsar712/stacklift/internal/config"
	"github.com/lacsar712/stacklift/internal/crane"
	"github.com/lacsar712/stacklift/internal/fsm"
	"github.com/lacsar712/stacklift/internal/interlock"
	"github.com/lacsar712/stacklift/internal/model"
	"github.com/lacsar712/stacklift/internal/path"
	"github.com/lacsar712/stacklift/internal/safety"
	"github.com/lacsar712/stacklift/internal/store"
)

type App struct {
	mu        sync.RWMutex
	cfg       config.Config
	clk       *clock.DualClock
	coord     *crane.Coordinator
	rig       *store.Rig
	fsm       *fsm.MotionMachine
	guard     *interlock.Guard
	estop     *safety.EStopController
	registry  *aisle.Registry
	occupancy *aisle.Occupancy
	planner   *path.Planner
	started   bool
}

func New(cfg config.Config) (*App, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("app init: %w", err)
	}
	clk := clock.NewDual(cfg.ClockStep)
	layout := config.NewLayout(cfg.Warehouse)
	planner := path.NewPlanner(layout, cfg.Limits)
	coord := crane.NewCoordinator(planner, clk, cfg.Limits)
	occupancy := aisle.NewOccupancy(layout.AllCells())
	monitor := safety.NewMonitor(cfg.Limits)
	rig := store.NewRig(coord, occupancy, monitor, clk)
	guard := interlock.NewGuard(cfg.Limits, clk)
	estop := safety.NewEStopController(coord, monitor)
	machine := fsm.NewMotionMachine()
	fsm.RegisterDefaultEffects(machine, coord, clk, cfg.Limits)
	return &App{
		cfg: cfg, clk: clk, coord: coord, rig: rig, fsm: machine,
		guard: guard, estop: estop, registry: aisle.NewRegistry(),
		occupancy: occupancy, planner: planner,
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.started {
		return nil
	}
	layout := config.NewLayout(a.cfg.Warehouse)
	for i := 1; i <= a.cfg.Warehouse.Aisles; i++ {
		aisleID := model.AisleID(fmt.Sprintf("%02d", i))
		craneID := model.CraneID(fmt.Sprintf("CR-%02d", i))
		svc := a.coord.Register(craneID)
		home := layout.CraneHome(aisleID)
		pos := layout.LocationToPosition(home)
		svc.SetPose(model.CranePose{
			CraneID: craneID, Location: home,
			TravelMM: pos.TravelMM, HoistMM: pos.HoistMM, ForkMM: pos.ForkMM,
			ForkPos: model.ForkRetracted,
		})
		a.fsm.Ensure(craneID)
		_ = a.registry.Register(aisle.AisleInfo{
			ID: aisleID, CraneID: craneID,
			LengthMM: int64(a.cfg.Warehouse.BaysPerAisle) * a.cfg.Warehouse.BayPitchMM, Active: true,
		})
	}
	a.started = true
	a.rig.Capture()
	return nil
}

func (a *App) Config() config.Config              { return a.cfg }
func (a *App) Clock() *clock.DualClock           { return a.clk }
func (a *App) Coordinator() *crane.Coordinator   { return a.coord }
func (a *App) Rig() *store.Rig                   { return a.rig }
func (a *App) FSM() *fsm.MotionMachine           { return a.fsm }
func (a *App) Guard() *interlock.Guard           { return a.guard }
func (a *App) EStop() *safety.EStopController    { return a.estop }
func (a *App) Occupancy() *aisle.Occupancy       { return a.occupancy }
func (a *App) Registry() *aisle.Registry           { return a.registry }
func (a *App) Planner() *path.Planner            { return a.planner }

func (a *App) Snapshot() model.WarehouseSnapshot { return a.rig.Capture() }

func (a *App) MoveCrane(ctx context.Context, craneID model.CraneID, from, to model.Location) error {
	if err := a.guard.AcquireHoistLease(craneID); err != nil {
		return err
	}
	defer a.guard.ReleaseHoistLease(craneID)
	svc, ok := a.coord.Get(craneID)
	if !ok {
		return model.ErrCraneNotFound
	}
	status := svc.Status()
	plan, err := a.planner.Plan("", craneID, from, to)
	if err != nil {
		return err
	}
	if err := a.estop.Monitor().CheckTravelSpeed(craneID, a.planner.PeakTravelSpeed(plan)); err != nil {
		return fmt.Errorf("motion: %w", err)
	}
	for _, step := range plan.Steps {
		if err := a.guard.ValidateMotion(ctx, status, step.Axis); err != nil {
			return err
		}
	}
	if err := a.coord.MoveCrane(ctx, craneID, from, to); err != nil {
		if errors.Is(err, model.ErrEncoderSlip) {
			return crane.WrapMotionFault(err)
		}
		return err
	}
	a.rig.RefreshCrane(craneID)
	return nil
}

func (a *App) Shutdown(ctx context.Context) {
	a.coord.CancelAll()
	a.started = false
}

func (a *App) Started() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.started
}

func (a *App) CraneCount() int { return a.coord.Count() }

func (a *App) AdvanceClock(n int64) int64 { return a.clk.Process.Advance(n) }
