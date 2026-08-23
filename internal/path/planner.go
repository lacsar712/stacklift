package path

import (
	"fmt"

	"github.com/lacsar712/stacklift/internal/config"
	"github.com/lacsar712/stacklift/internal/model"
)

type Planner struct {
	layout *config.WarehouseLayout
	limits model.LimitSet
}

func NewPlanner(layout *config.WarehouseLayout, limits model.LimitSet) *Planner {
	return &Planner{layout: layout, limits: limits}
}

func (p *Planner) Plan(taskID model.TaskID, crane model.CraneID, from, to model.Location) (model.MotionPlan, error) {
	if !p.layout.InBounds(from) || !p.layout.InBounds(to) {
		return model.MotionPlan{}, model.ErrInvalidLocation
	}
	if from.Aisle != to.Aisle {
		return model.MotionPlan{}, fmt.Errorf("cross-aisle move not supported: %w", model.ErrInvalidLocation)
	}
	fromPos := p.layout.LocationToPosition(from)
	toPos := p.layout.LocationToPosition(to)
	steps := p.buildSteps(fromPos, toPos)
	plan := model.MotionPlan{TaskID: taskID, CraneID: crane, From: fromPos, To: toPos, Steps: steps}
	var total int64
	for _, s := range steps {
		total += s.DurationMS()
	}
	plan.EstimatedMS = total
	return plan, nil
}

func (p *Planner) buildSteps(from, to model.Position) []model.MotionStep {
	var steps []model.MotionStep
	order := 1
	if from.TravelMM != to.TravelMM {
		steps = append(steps, model.MotionStep{Axis: model.AxisTravel, FromMM: from.TravelMM, ToMM: to.TravelMM, SpeedMMS: p.limits.MaxTravelSpeed, Order: order})
		order++
	}
	if from.HoistMM != to.HoistMM {
		steps = append(steps, model.MotionStep{Axis: model.AxisHoist, FromMM: from.HoistMM, ToMM: to.HoistMM, SpeedMMS: p.limits.MaxHoistSpeed, Order: order})
		order++
	}
	if from.ForkMM != to.ForkMM {
		steps = append(steps, model.MotionStep{Axis: model.AxisFork, FromMM: from.ForkMM, ToMM: to.ForkMM, SpeedMMS: p.limits.MaxForkSpeed, Order: order})
	}
	return steps
}

func (p *Planner) PlanRetrieval(task model.RetrievalTask, current model.Location) (model.MotionPlan, error) {
	return p.Plan(task.ID, task.CraneID, current, task.Source)
}

func (p *Planner) PlanDeposit(task model.RetrievalTask, current model.Location) (model.MotionPlan, error) {
	return p.Plan(task.ID, task.CraneID, current, task.Dest)
}

func (p *Planner) EstimateDistance(from, to model.Location) int64 {
	return model.DistanceMM(p.layout.LocationToPosition(from), p.layout.LocationToPosition(to))
}
