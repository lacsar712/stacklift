package model_test

import (
	"testing"

	"github.com/lacsar712/stacklift/internal/model"
)

func TestClone(t *testing.T) {
	st := model.RigStatus{RigID: "TC-01", Mode: model.DutyIdle}
	cp := st.Clone()
	cp.Mode = model.DutyEmergency
	if st.Mode == model.DutyEmergency {
		t.Fatal("alias")
	}
}

func TestDeepCopyPose(t *testing.T) {
	src := []model.RigPose{{RigID: "a", AzimuthDeg: 1}}
	cp := model.DeepCopyPose(src)
	cp[0].AzimuthDeg = 9
	if src[0].AzimuthDeg == 9 {
		t.Fatal("alias")
	}
}
