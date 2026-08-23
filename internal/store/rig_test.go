package store_test

import (
	"testing"

	"github.com/lacsar712/stacklift/internal/model"
	"github.com/lacsar712/stacklift/internal/store"
)

func TestSnapshotDeepCopy(t *testing.T) {
	s := store.NewRigStore()
	s.PutPose(model.RigPose{RigID: "TC-01", AzimuthDeg: 10})
	snap := s.Snapshot()
	snap[0].AzimuthDeg = 99
	p, _ := s.GetPose("TC-01")
	if p.AzimuthDeg == 99 {
		t.Fatal("alias")
	}
}
