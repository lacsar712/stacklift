package aisle

import (
	"fmt"
	"sync"

	"github.com/lacsar712/stacklift/internal/model"
)

type Occupancy struct {
	mu    sync.RWMutex
	cells map[string]model.StorageCell
}

func cellKey(loc model.Location) string {
	return fmt.Sprintf("%s:%d:%d:%d", loc.Aisle, loc.Bay, loc.Level, loc.Depth)
}

func NewOccupancy(cells []model.StorageCell) *Occupancy {
	o := &Occupancy{cells: make(map[string]model.StorageCell, len(cells))}
	for _, c := range cells {
		o.cells[cellKey(c.Location)] = c
	}
	return o
}

func (o *Occupancy) Get(loc model.Location) (model.StorageCell, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	c, ok := o.cells[cellKey(loc)]
	return c.Clone(), ok
}

func (o *Occupancy) IsOccupied(loc model.Location) bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	c, ok := o.cells[cellKey(loc)]
	return ok && c.Occupied
}

func (o *Occupancy) IsReserved(loc model.Location) bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	c, ok := o.cells[cellKey(loc)]
	return ok && c.Reserved
}

func (o *Occupancy) Place(loc model.Location, pallet model.PalletID) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	key := cellKey(loc)
	c, ok := o.cells[key]
	if !ok {
		return model.ErrInvalidLocation
	}
	if c.Occupied {
		return model.ErrCellOccupied
	}
	if c.Reserved && c.ReservedBy != "" {
		return model.ErrCellReserved
	}
	c.Occupied = true
	c.PalletID = pallet
	c.Reserved = false
	c.ReservedBy = ""
	o.cells[key] = c
	return nil
}

func (o *Occupancy) Remove(loc model.Location) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	key := cellKey(loc)
	c, ok := o.cells[key]
	if !ok {
		return model.ErrInvalidLocation
	}
	if !c.Occupied {
		return model.ErrCellEmpty
	}
	c.Occupied = false
	c.PalletID = ""
	o.cells[key] = c
	return nil
}

func (o *Occupancy) Reserve(loc model.Location, crane model.CraneID) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	key := cellKey(loc)
	c, ok := o.cells[key]
	if !ok {
		return model.ErrInvalidLocation
	}
	if c.Reserved && c.ReservedBy != crane {
		return model.ErrCellReserved
	}
	c.Reserved = true
	c.ReservedBy = crane
	o.cells[key] = c
	return nil
}

func (o *Occupancy) Release(loc model.Location, crane model.CraneID) {
	o.mu.Lock()
	defer o.mu.Unlock()
	key := cellKey(loc)
	c, ok := o.cells[key]
	if !ok {
		return
	}
	if c.Reserved && c.ReservedBy == crane {
		c.Reserved = false
		c.ReservedBy = ""
		o.cells[key] = c
	}
}

func (o *Occupancy) Snapshot() []model.StorageCell {
	o.mu.RLock()
	defer o.mu.RUnlock()
	out := make([]model.StorageCell, 0, len(o.cells))
	for _, c := range o.cells {
		out = append(out, c.Clone())
	}
	return out
}

func (o *Occupancy) FindPallet(pallet model.PalletID) (model.Location, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	for _, c := range o.cells {
		if c.Occupied && c.PalletID == pallet {
			return c.Location, true
		}
	}
	return model.Location{}, false
}

func (o *Occupancy) CountOccupied() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	n := 0
	for _, c := range o.cells {
		if c.Occupied {
			n++
		}
	}
	return n
}
