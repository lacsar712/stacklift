package crane

import (
	"context"
	"fmt"

	"github.com/lacsar712/stacklift/internal/model"
	"github.com/lacsar712/stacklift/internal/path"
)

type MotionExecutor struct {
	service *Service
	route   *path.Route
}

func NewMotionExecutor(service *Service, route *path.Route) *MotionExecutor {
	return &MotionExecutor{service: service, route: route}
}

func (e *MotionExecutor) Execute(ctx context.Context) error {
	for {
		step, ok := e.route.Current()
		if !ok {
			return nil
		}
		stepCtx := ctx
		if step.Axis == model.AxisHoist {
			stepCtx = context.WithoutCancel(ctx)
		}
		if err := e.service.MoveAxis(stepCtx, step.Axis, step.ToMM); err != nil {
			return fmt.Errorf("step %d axis %s: %w", e.route.CompletedSteps(), step.Axis, err)
		}
		e.route.Advance()
	}
}

func (e *MotionExecutor) ExecuteOne(ctx context.Context) error {
	step, ok := e.route.Current()
	if !ok {
		return nil
	}
	if err := e.service.MoveAxis(ctx, step.Axis, step.ToMM); err != nil {
		return err
	}
	e.route.Advance()
	return nil
}

func (e *MotionExecutor) Remaining() int       { return len(e.route.Remaining()) }
func (e *MotionExecutor) Progress() float64    { return e.route.Progress() }
func (e *MotionExecutor) Route() *path.Route   { return e.route }
func (e *MotionExecutor) Cancel()              { e.service.CancelMotion() }

func ExecutePlan(ctx context.Context, service *Service, plan model.MotionPlan) error {
	route := path.NewRoute(plan)
	exec := NewMotionExecutor(service, route)
	return exec.Execute(ctx)
}
