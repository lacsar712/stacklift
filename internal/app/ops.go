package app

import (
	"context"
	"fmt"

	"github.com/lacsar712/stacklift/internal/config"
	"github.com/lacsar712/stacklift/internal/fsm"
	"github.com/lacsar712/stacklift/internal/model"
)

func RunDemo(ctx context.Context, a *App) error {
	if err := a.Start(ctx); err != nil {
		return err
	}
	cranes := a.Coordinator().CraneIDs()
	if len(cranes) == 0 {
		return fmt.Errorf("no cranes registered")
	}
	craneID := cranes[0]
	aisleID, ok := a.Registry().AisleForCrane(craneID)
	if !ok {
		aisleID = model.AisleID("01")
	}
	home := model.Location{Aisle: aisleID, Bay: 1, Level: 1, Depth: 0}
	target := model.Location{Aisle: aisleID, Bay: 3, Level: 2, Depth: 1}
	if err := a.Occupancy().Place(target, model.PalletID("PLT-001")); err != nil {
		return err
	}
	machine := a.FSM()
	ctx = fsm.WithDwellMachine(ctx, machine)
	if _, err := machine.Fire(ctx, craneID, model.EventStartTravel); err != nil {
		return err
	}
	if _, err := machine.Fire(ctx, craneID, model.EventTravelDone); err != nil {
		return err
	}
	if err := a.MoveCrane(ctx, craneID, home, target); err != nil {
		return err
	}
	a.AdvanceClock(a.Config().Limits.DwellWindowTicks)
	if _, err := machine.Fire(ctx, craneID, model.EventDwellStart); err == nil {
		a.Guard().StartDwell(craneID)
		a.AdvanceClock(a.Config().Limits.DwellWindowTicks)
		_, _ = machine.Fire(ctx, craneID, model.EventDwellEnd)
	}
	snap := a.Snapshot()
	if len(snap.Cranes) == 0 {
		return fmt.Errorf("empty snapshot")
	}
	return nil
}

func LoadConfig() (config.Config, error) { return config.LoadFromEnv() }

func NewDefault() (*App, error) { return New(config.Default()) }

func (a *App) PlacePallet(loc model.Location, pallet model.PalletID) error {
	if err := a.Occupancy().Place(loc, pallet); err != nil {
		return err
	}
	a.Rig().RefreshCells()
	return nil
}

func (a *App) RetrievePallet(ctx context.Context, craneID model.CraneID, pallet model.PalletID, dest model.Location) error {
	source, ok := a.Occupancy().FindPallet(pallet)
	if !ok {
		return model.ErrTaskNotFound
	}
	svc, ok := a.Coordinator().Get(craneID)
	if !ok {
		return model.ErrCraneNotFound
	}
	current := svc.Pose().Location
	if err := a.MoveCrane(ctx, craneID, current, source); err != nil {
		return err
	}
	if err := svc.LoadPallet(pallet); err != nil {
		return err
	}
	if err := a.Occupancy().Remove(source); err != nil {
		return err
	}
	if err := a.MoveCrane(ctx, craneID, source, dest); err != nil {
		return err
	}
	if _, err := svc.UnloadPallet(); err != nil {
		return err
	}
	return a.Occupancy().Place(dest, pallet)
}
