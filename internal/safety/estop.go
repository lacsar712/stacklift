package safety

import (
	"sync"

	"github.com/lacsar712/stacklift/internal/crane"
	"github.com/lacsar712/stacklift/internal/model"
)

type EStopController struct {
	mu        sync.RWMutex
	active    bool
	coord     *crane.Coordinator
	monitor   *Monitor
	triggered []model.CraneID
}

func NewEStopController(coord *crane.Coordinator, monitor *Monitor) *EStopController {
	return &EStopController{coord: coord, monitor: monitor}
}

func (e *EStopController) Trigger() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.active = true
	e.monitor.SetEStop(true)
	e.coord.SetEStopAll(true)
	e.coord.CancelAll()
	e.triggered = e.coord.CraneIDs()
}

func (e *EStopController) Reset() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.active = false
	e.monitor.SetEStop(false)
	for _, id := range e.coord.CraneIDs() {
		svc, ok := e.coord.Get(id)
		if !ok {
			continue
		}
		svc.SetEStop(false)
		svc.SetPhase(model.PhaseIdle)
	}
	e.triggered = nil
	return nil
}

func (e *EStopController) Active() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.active
}

func (e *EStopController) TriggeredCranes() []model.CraneID {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]model.CraneID, len(e.triggered))
	copy(out, e.triggered)
	return out
}

func (e *EStopController) Monitor() *Monitor { return e.monitor }
