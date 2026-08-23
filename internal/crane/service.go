package crane

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/lacsar712/stacklift/internal/clock"
	"github.com/lacsar712/stacklift/internal/fork"
	"github.com/lacsar712/stacklift/internal/model"
)

func WrapMotionFault(err error) error {
	if errors.Is(err, model.ErrEncoderSlip) {
		return fmt.Errorf("motion fault: %w", err)
	}
	return err
}

type Service struct {
	mu          sync.RWMutex
	craneID     model.CraneID
	status      model.CraneStatus
	limits      model.LimitSet
	forkHandler *fork.LoadHandler
	clk         *clock.DualClock
	hoistLocked bool
	cancelFn    context.CancelFunc
}

func NewService(id model.CraneID, limits model.LimitSet, clk *clock.DualClock) *Service {
	return &Service{
		craneID: id, status: model.EmptyCraneStatus(id), limits: limits,
		forkHandler: fork.NewLoadHandler(id, limits.MaxForkMM), clk: clk,
	}
}

func (s *Service) ID() model.CraneID { return s.craneID }

func (s *Service) Status() model.CraneStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status.Clone()
}

func (s *Service) SetPose(pose model.CranePose) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pose.CraneID = s.craneID
	s.status.Pose = pose.Clone()
	s.status.Revision++
}

func (s *Service) SetPhase(phase model.CranePhase) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status.Phase = phase
	s.status.Revision++
	s.status.UpdatedAt = s.clk.WallNow()
}

func (s *Service) Pose() model.CranePose {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status.Pose.Clone()
}

func (s *Service) ForkHandler() *fork.LoadHandler { return s.forkHandler }

func (s *Service) HoistLocked() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hoistLocked
}

func (s *Service) lockHoist() {
	s.mu.Lock()
	s.hoistLocked = true
	s.mu.Unlock()
}

func (s *Service) unlockHoist() {
	s.mu.Lock()
	s.hoistLocked = false
	s.mu.Unlock()
}

func (s *Service) MoveAxis(ctx context.Context, axis model.MotionAxis, targetMM int64) error {
	if err := s.checkMotionAllowed(axis); err != nil {
		return err
	}
	motionCtx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	if s.cancelFn != nil {
		s.cancelFn()
	}
	s.cancelFn = cancel
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.cancelFn = nil
		s.mu.Unlock()
	}()
	if axis == model.AxisHoist {
		s.lockHoist()
		defer s.unlockHoist()
	}
	select {
	case <-motionCtx.Done():
		return fmt.Errorf("move %s: %w", axis, model.ErrContextDone)
	default:
	}
	return s.executeMotion(motionCtx, axis, targetMM)
}

func (s *Service) checkMotionAllowed(axis model.MotionAxis) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.status.EStopActive {
		return model.ErrEStopActive
	}
	if s.status.Interlocked {
		return model.NewInterlockError("service", s.craneID, "crane interlocked")
	}
	if model.IsMotionPhase(s.status.Phase) {
		active, ok := model.ActiveAxis(s.status.Phase)
		if ok && active != axis {
			return model.NewMotionError(axis, s.craneID, model.ErrAxisConflict)
		}
	}
	return nil
}

func (s *Service) executeMotion(ctx context.Context, axis model.MotionAxis, targetMM int64) error {
	s.mu.Lock()
	s.status.Phase = phaseForAxis(axis)
	progress := model.MotionProgress{
		Axis: axis, CurrentMM: s.currentMMLocked(axis), TargetMM: targetMM,
		StartedTick: s.clk.ProcessTick(),
	}
	s.setProgressLocked(axis, progress)
	s.status.Revision++
	s.mu.Unlock()

	select {
	case <-ctx.Done():
		return fmt.Errorf("execute %s: %w", axis, model.ErrContextDone)
	default:
	}

	if axis == model.AxisTravel && targetMM > s.status.Pose.TravelMM+5000 {
		return model.ErrEncoderSlip
	}
	s.mu.Lock()
	s.setMMLocked(axis, targetMM)
	progress.Complete = true
	progress.CurrentMM = targetMM
	s.setProgressLocked(axis, progress)
	s.status.Phase = model.PhaseIdle
	s.status.Pose.ProcessTick = s.clk.ProcessTick()
	s.status.UpdatedAt = s.clk.WallNow()
	s.status.Revision++
	s.mu.Unlock()
	return nil
}

func phaseForAxis(axis model.MotionAxis) model.CranePhase {
	switch axis {
	case model.AxisTravel:
		return model.PhaseTraveling
	case model.AxisHoist:
		return model.PhaseHoisting
	case model.AxisFork:
		return model.PhaseForking
	default:
		return model.PhaseIdle
	}
}

func (s *Service) currentMMLocked(axis model.MotionAxis) int64 {
	switch axis {
	case model.AxisTravel:
		return s.status.Pose.TravelMM
	case model.AxisHoist:
		return s.status.Pose.HoistMM
	case model.AxisFork:
		return s.status.Pose.ForkMM
	default:
		return 0
	}
}

func (s *Service) setMMLocked(axis model.MotionAxis, mm int64) {
	switch axis {
	case model.AxisTravel:
		s.status.Pose.TravelMM = mm
	case model.AxisHoist:
		s.status.Pose.HoistMM = mm
	case model.AxisFork:
		s.status.Pose.ForkMM = mm
	}
}

func (s *Service) setProgressLocked(axis model.MotionAxis, p model.MotionProgress) {
	switch axis {
	case model.AxisTravel:
		s.status.Travel = p
	case model.AxisHoist:
		s.status.Hoist = p
	case model.AxisFork:
		s.status.Fork = p
	}
}

func (s *Service) CancelMotion() {
	s.mu.Lock()
	fn := s.cancelFn
	s.mu.Unlock()
	if fn != nil {
		fn()
	}
}

func (s *Service) SetEStop(active bool) {
	s.mu.Lock()
	s.status.EStopActive = active
	if active {
		s.status.Phase = model.PhaseEmergency
	}
	s.status.Revision++
	fn := s.cancelFn
	if active && fn != nil {
		s.cancelFn = nil
	}
	s.mu.Unlock()
	if active && fn != nil {
		// Propagate the aisle estop button into the motion execution layer:
		// cancel any in-flight axis motion so the executor stops emitting
		// velocity commands from cached route segments instead of waiting
		// for the move to drain on its own. Mirrors the FSM estop path,
		// which calls CancelMotion() alongside SetEStop(true).
		fn()
	}
}

func (s *Service) SetInterlocked(locked bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status.Interlocked = locked
	s.status.Revision++
}

func (s *Service) LoadPallet(pallet model.PalletID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status.Pose.Loaded {
		return fmt.Errorf("already loaded: %w", model.ErrMotionInProgress)
	}
	s.status.Pose.Loaded = true
	s.status.Pose.PalletID = pallet
	s.status.Revision++
	return nil
}

func (s *Service) UnloadPallet() (model.PalletID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.status.Pose.Loaded {
		return "", fmt.Errorf("not loaded: %w", model.ErrCellEmpty)
	}
	id := s.status.Pose.PalletID
	s.status.Pose.Loaded = false
	s.status.Pose.PalletID = ""
	s.status.Revision++
	return id, nil
}

func (s *Service) IsContextError(err error) bool {
	return errors.Is(err, model.ErrContextDone)
}
