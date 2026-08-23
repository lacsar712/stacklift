package path

import "github.com/lacsar712/stacklift/internal/model"

type Route struct {
	Plan   model.MotionPlan
	Cursor int
	Done   bool
}

func NewRoute(plan model.MotionPlan) *Route {
	return &Route{Plan: plan.Clone(), Cursor: 0}
}

func (r *Route) Current() (model.MotionStep, bool) {
	if r.Done || r.Cursor >= len(r.Plan.Steps) {
		return model.MotionStep{}, false
	}
	return r.Plan.Steps[r.Cursor], true
}

func (r *Route) Advance() {
	r.Cursor++
	if r.Cursor >= len(r.Plan.Steps) {
		r.Done = true
	}
}

func (r *Route) Remaining() []model.MotionStep {
	if r.Done {
		return nil
	}
	return r.Plan.Steps[r.Cursor:]
}

func (r *Route) Progress() float64 {
	if len(r.Plan.Steps) == 0 {
		return 1.0
	}
	return float64(r.Cursor) / float64(len(r.Plan.Steps))
}

func (r *Route) TotalSteps() int     { return len(r.Plan.Steps) }
func (r *Route) CompletedSteps() int { return r.Cursor }

func (r *Route) Reset() {
	r.Cursor = 0
	r.Done = false
}

func (r *Route) Clone() *Route {
	return &Route{Plan: r.Plan.Clone(), Cursor: r.Cursor, Done: r.Done}
}

func (r *Route) AxisSequence() []model.MotionAxis {
	axes := make([]model.MotionAxis, len(r.Plan.Steps))
	for i, s := range r.Plan.Steps {
		axes[i] = s.Axis
	}
	return axes
}

func (r *Route) HasAxis(axis model.MotionAxis) bool {
	for _, s := range r.Plan.Steps {
		if s.Axis == axis {
			return true
		}
	}
	return false
}
