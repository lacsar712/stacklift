package fork

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/lacsar712/stacklift/internal/model"
)

var ErrForkJam = errors.New("fork jam detected")

type Actuator struct {
	mu          sync.Mutex
	craneID     model.CraneID
	position    model.ForkPosition
	extensionMM int64
	maxMM       int64
	moving      bool
}

func NewActuator(crane model.CraneID, maxMM int64) *Actuator {
	return &Actuator{craneID: crane, position: model.ForkRetracted, maxMM: maxMM}
}

func (a *Actuator) Position() model.ForkPosition {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.position
}

func (a *Actuator) ExtensionMM() int64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.extensionMM
}

func (a *Actuator) IsMoving() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.moving
}

func (a *Actuator) Extend(ctx context.Context, targetMM int64) error {
	return a.move(ctx, targetMM, model.ForkExtended)
}

func (a *Actuator) Retract(ctx context.Context) error {
	return a.move(ctx, 0, model.ForkRetracted)
}

func (a *Actuator) move(ctx context.Context, targetMM int64, pos model.ForkPosition) error {
	a.mu.Lock()
	if a.moving {
		a.mu.Unlock()
		return model.NewMotionError(model.AxisFork, a.craneID, model.ErrMotionInProgress)
	}
	if targetMM < 0 || targetMM > a.maxMM {
		a.mu.Unlock()
		return model.NewMotionError(model.AxisFork, a.craneID, fmt.Errorf("target %d out of range", targetMM))
	}
	a.moving = true
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.moving = false
		a.mu.Unlock()
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("fork move: %w", model.ErrContextDone)
	default:
	}

	a.mu.Lock()
	a.extensionMM = targetMM
	a.position = pos
	if targetMM == 0 {
		a.position = model.ForkRetracted
	} else if targetMM >= a.maxMM/2 {
		a.position = model.ForkExtended
	} else {
		a.position = model.ForkCentered
	}
	a.mu.Unlock()
	return nil
}

func (a *Actuator) Center(ctx context.Context) error {
	return a.move(ctx, a.maxMM/2, model.ForkCentered)
}

func (a *Actuator) IsExtended() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.position == model.ForkExtended
}

func (a *Actuator) IsRetracted() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.position == model.ForkRetracted
}

func (a *Actuator) Snapshot() (model.ForkPosition, int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.position, a.extensionMM
}

type Telescope struct {
	left  *Actuator
	right *Actuator
	mu    sync.Mutex
}

func NewTelescope(crane model.CraneID, maxMM int64) *Telescope {
	return &Telescope{left: NewActuator(crane, maxMM), right: NewActuator(crane, maxMM)}
}

func (t *Telescope) CheckLoadBalance(maxDeltaMM int64) error {
	_, leftMM := t.left.Snapshot()
	_, rightMM := t.right.Snapshot()
	delta := leftMM - rightMM
	if delta < 0 { delta = -delta }
	if delta > maxDeltaMM {
		return fmt.Errorf("fork timeout: %w", context.DeadlineExceeded)
	}
	return nil
}

func (t *Telescope) ExtendBoth(ctx context.Context, targetMM int64) error {
	if err := t.CheckLoadBalance(50); err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if err := t.left.Extend(ctx, targetMM); err != nil {
		return err
	}
	return t.right.Extend(ctx, targetMM)
}

func (t *Telescope) RetractBoth(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if err := t.left.Retract(ctx); err != nil {
		return err
	}
	return t.right.Retract(ctx)
}

func (t *Telescope) Left() *Actuator  { return t.left }
func (t *Telescope) Right() *Actuator { return t.right }

func (t *Telescope) BothRetracted() bool {
	return t.left.IsRetracted() && t.right.IsRetracted()
}
