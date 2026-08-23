package alarm_test

import (
	"testing"

	"github.com/lacsar712/stacklift/internal/alarm"
	"github.com/lacsar712/stacklift/internal/model"
)

func TestBus(t *testing.T) {
	b := alarm.NewBus(0)
	b.Raise("TC-01", "T", model.AlarmWarn, "m")
	if b.CountActive() != 1 {
		t.Fatal("active")
	}
}
