package fork

import (
	"context"

	"github.com/lacsar712/stacklift/internal/model"
)

type LoadHandler struct {
	telescope *Telescope
	craneID   model.CraneID
}

func NewLoadHandler(crane model.CraneID, maxMM int64) *LoadHandler {
	return &LoadHandler{telescope: NewTelescope(crane, maxMM), craneID: crane}
}

func (h *LoadHandler) Pickup(ctx context.Context, depthMM int64) error {
	if !h.telescope.BothRetracted() {
		if err := h.telescope.RetractBoth(ctx); err != nil {
			return err
		}
	}
	return h.telescope.ExtendBoth(ctx, depthMM)
}

func (h *LoadHandler) Deposit(ctx context.Context, depthMM int64) error {
	if err := h.telescope.ExtendBoth(ctx, depthMM); err != nil {
		return err
	}
	return h.telescope.RetractBoth(ctx)
}

func (h *LoadHandler) Telescope() *Telescope { return h.telescope }

func (h *LoadHandler) Status() (model.ForkPosition, int64) {
	return h.telescope.Left().Snapshot()
}

func (h *LoadHandler) CraneID() model.CraneID { return h.craneID }
