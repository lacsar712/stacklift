package counter_test

import (
	"errors"
	"testing"
	"time"

	"github.com/lacsar712/stacklift/internal/counter"
)

func TestLimit(t *testing.T) {
	c := counter.New(1)
	_ = c.Increment("TC-01")
	err := c.Increment("TC-01")
	if !errors.Is(err, counter.ErrCycleLimit) {
		t.Fatal(err)
	}
}

func TestStale(t *testing.T) {
	c := counter.New(10)
	if !errors.Is(c.Check("TC-01", time.Now()), counter.ErrCounterStale) {
		t.Fatal("stale")
	}
}
