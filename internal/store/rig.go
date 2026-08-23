package store

import (
	"fmt"
	"sync"
	"time"

	"github.com/lacsar712/stacklift/internal/model"
)

type RigStore struct {
	mu     sync.RWMutex
	poses  map[string]model.RigPose
	status map[string]model.RigStatus
	rev    uint64
}

func NewRigStore() *RigStore {
	return &RigStore{poses: make(map[string]model.RigPose), status: make(map[string]model.RigStatus)}
}

func (s *RigStore) PutPose(pose model.RigPose) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rev++
	s.poses[pose.RigID] = pose.Clone()
}

func (s *RigStore) GetPose(rigID string) (model.RigPose, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.poses[rigID]
	if !ok {
		return model.RigPose{}, false
	}
	return p.Clone(), true
}

func (s *RigStore) Snapshot() []model.RigPose {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.RigPose, 0, len(s.poses))
	for _, p := range s.poses {
		out = append(out, p.Clone())
	}
	return out
}

func (s *RigStore) PutStatus(st model.RigStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rev++
	cp := st.Clone()
	cp.Revision = s.rev
	s.status[st.RigID] = cp
}

func (s *RigStore) StatusSnapshot() []model.RigStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.RigStatus, 0, len(s.status))
	for _, st := range s.status {
		out = append(out, st.Clone())
	}
	return out
}

func (s *RigStore) UpdatePose(rigID string, fn func(*model.RigPose) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.poses[rigID]
	if !ok {
		return fmt.Errorf("store: unknown rig %s", rigID)
	}
	cp := p.Clone()
	if err := fn(&cp); err != nil {
		return err
	}
	s.rev++
	cp.UpdatedAt = time.Now().UTC()
	s.poses[rigID] = cp
	return nil
}

func (s *RigStore) Revision() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rev
}
