package config

import (
	"fmt"

	"github.com/lacsar712/stacklift/internal/model"
)

type WarehouseLayout struct {
	cfg WarehouseConfig
}

func NewLayout(cfg WarehouseConfig) *WarehouseLayout {
	return &WarehouseLayout{cfg: cfg}
}

func (l *WarehouseLayout) AllCells() []model.StorageCell {
	total := l.cfg.Aisles * l.cfg.BaysPerAisle * l.cfg.Levels * (l.cfg.Depths + 1)
	cells := make([]model.StorageCell, 0, total)
	for a := 1; a <= l.cfg.Aisles; a++ {
		aisle := model.AisleID(fmt.Sprintf("%02d", a))
		for bay := 1; bay <= l.cfg.BaysPerAisle; bay++ {
			for level := 1; level <= l.cfg.Levels; level++ {
				for depth := 0; depth <= l.cfg.Depths; depth++ {
					cells = append(cells, model.StorageCell{
						Location: model.Location{Aisle: aisle, Bay: bay, Level: level, Depth: depth},
					})
				}
			}
		}
	}
	return cells
}

func (l *WarehouseLayout) LocationToPosition(loc model.Location) model.Position {
	return model.LocationToPosition(loc, l.cfg.BayPitchMM, l.cfg.LevelPitchMM, l.cfg.DepthPitchMM)
}

func (l *WarehouseLayout) PositionToLocation(pos model.Position, aisle model.AisleID) model.Location {
	return model.PositionToLocation(pos, aisle, l.cfg.BayPitchMM, l.cfg.LevelPitchMM, l.cfg.DepthPitchMM)
}

func (l *WarehouseLayout) InBounds(loc model.Location) bool {
	if loc.Bay < 1 || loc.Bay > l.cfg.BaysPerAisle {
		return false
	}
	if loc.Level < 1 || loc.Level > l.cfg.Levels {
		return false
	}
	if loc.Depth < 0 || loc.Depth > l.cfg.Depths {
		return false
	}
	return loc.Aisle != ""
}

func (l *WarehouseLayout) Config() WarehouseConfig { return l.cfg }

func (l *WarehouseLayout) CraneHome(aisle model.AisleID) model.Location {
	return model.Location{Aisle: aisle, Bay: 1, Level: 1, Depth: 0}
}

func (l *WarehouseLayout) MaxTravelMM() int64 {
	return int64(l.cfg.BaysPerAisle) * l.cfg.BayPitchMM
}

func (l *WarehouseLayout) MaxHoistMM() int64 {
	return int64(l.cfg.Levels) * l.cfg.LevelPitchMM
}
