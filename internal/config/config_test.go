package config_test

import (
	"testing"

	"github.com/lacsar712/stacklift/internal/config"
)

func TestDefaultValid(t *testing.T) {
	if err := config.Default().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestWindHold(t *testing.T) {
	cfg := config.Default()
	if cfg.WindHoldDuration() <= 0 {
		t.Fatal("wind hold duration")
	}
}
