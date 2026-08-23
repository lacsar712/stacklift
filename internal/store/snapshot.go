package store

import (
	"sync"

	"github.com/lacsar712/stacklift/internal/model"
)

type SnapshotStore struct {
	mu       sync.RWMutex
	current  model.WarehouseSnapshot
	revision uint64
}

func NewSnapshotStore() *SnapshotStore {
	return &SnapshotStore{revision: 0}
}

func (s *SnapshotStore) Save(snap model.WarehouseSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revision++
	snap.Seq = s.revision
	s.current = snap.Clone()
}

func (s *SnapshotStore) Load() model.WarehouseSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current.Clone()
}

func (s *SnapshotStore) Revision() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.revision
}

func (s *SnapshotStore) UpdateCrane(id model.CraneID, status model.CraneStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap := s.current.Clone()
	if snap.Cranes == nil {
		snap.Cranes = make(map[model.CraneID]model.CraneStatus)
	}
	snap.Cranes[id] = status.Clone()
	s.revision++
	snap.Seq = s.revision
	s.current = snap
}

func (s *SnapshotStore) UpdateCells(cells []model.StorageCell) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap := s.current.Clone()
	snap.Cells = model.DeepCopyCells(cells)
	s.revision++
	snap.Seq = s.revision
	s.current = snap
}

func (s *SnapshotStore) Crane(id model.CraneID) (model.CraneStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st, ok := s.current.Cranes[id]
	if !ok {
		return model.CraneStatus{}, false
	}
	return st.Clone(), true
}

func (s *SnapshotStore) IsStale(seq uint64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return seq < s.revision
}

func (s *SnapshotStore) CompareAndSwap(expected uint64, snap model.WarehouseSnapshot) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.revision != expected {
		return false
	}
	s.revision++
	snap.Seq = s.revision
	s.current = snap.Clone()
	return true
}
