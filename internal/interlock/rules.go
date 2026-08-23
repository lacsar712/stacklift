package interlock

import "github.com/lacsar712/stacklift/internal/model"

type Rule struct {
	Name        string
	Axis        model.MotionAxis
	Description string
	Check       func(status model.CraneStatus) error
}

type RuleSet struct {
	rules []Rule
}

func NewRuleSet(guard *Guard) *RuleSet {
	rs := &RuleSet{}
	rs.rules = []Rule{
		{Name: "travel_fork_retracted", Axis: model.AxisTravel, Description: "Fork must be retracted before travel", Check: guard.CheckTravel},
		{Name: "hoist_fork_clear", Axis: model.AxisHoist, Description: "Fork must not be extended during hoist", Check: guard.CheckHoist},
		{Name: "fork_no_travel", Axis: model.AxisFork, Description: "Fork motion blocked during travel", Check: guard.CheckFork},
	}
	return rs
}

func (rs *RuleSet) Rules() []Rule {
	out := make([]Rule, len(rs.rules))
	copy(out, rs.rules)
	return out
}

func (rs *RuleSet) CheckAll(status model.CraneStatus, axis model.MotionAxis) error {
	for _, r := range rs.rules {
		if r.Axis != axis {
			continue
		}
		if err := r.Check(status); err != nil {
			return err
		}
	}
	return nil
}

func (rs *RuleSet) RuleNames(axis model.MotionAxis) []string {
	var names []string
	for _, r := range rs.rules {
		if r.Axis == axis {
			names = append(names, r.Name)
		}
	}
	return names
}

func (rs *RuleSet) Count() int { return len(rs.rules) }

func (rs *RuleSet) CombinedCheck(status model.CraneStatus) error {
	for _, r := range rs.rules {
		if err := r.Check(status); err != nil {
			return err
		}
	}
	return nil
}
