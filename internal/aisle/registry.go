package aisle

import (
	"fmt"
	"sync"

	"github.com/lacsar712/stacklift/internal/model"
)

type Registry struct {
	mu     sync.RWMutex
	aisles map[model.AisleID]AisleInfo
}

type AisleInfo struct {
	ID       model.AisleID
	CraneID  model.CraneID
	LengthMM int64
	Active   bool
}

func NewRegistry() *Registry {
	return &Registry{aisles: make(map[model.AisleID]AisleInfo)}
}

func (r *Registry) Register(info AisleInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if info.ID == "" {
		return fmt.Errorf("aisle: empty id")
	}
	r.aisles[info.ID] = info
	return nil
}

func (r *Registry) Get(id model.AisleID) (AisleInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, ok := r.aisles[id]
	return info, ok
}

func (r *Registry) CraneForAisle(aisle model.AisleID) (model.CraneID, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, ok := r.aisles[aisle]
	if !ok || !info.Active {
		return "", false
	}
	return info.CraneID, true
}

func (r *Registry) AisleForCrane(crane model.CraneID) (model.AisleID, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for id, info := range r.aisles {
		if info.CraneID == crane && info.Active {
			return id, true
		}
	}
	return "", false
}

func (r *Registry) All() []AisleInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]AisleInfo, 0, len(r.aisles))
	for _, info := range r.aisles {
		out = append(out, info)
	}
	return out
}

func (r *Registry) Deactivate(id model.AisleID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if info, ok := r.aisles[id]; ok {
		info.Active = false
		r.aisles[id] = info
	}
}

func (r *Registry) Activate(id model.AisleID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if info, ok := r.aisles[id]; ok {
		info.Active = true
		r.aisles[id] = info
	}
}

func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.aisles)
}
