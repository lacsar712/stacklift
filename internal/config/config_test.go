package config_test

import (
	"testing"

	"github.com/lacsar712/stacklift/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	if err := config.Default().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestWarehouseLayout(t *testing.T) {
	cfg := config.DefaultWarehouse()
	layout := config.NewLayout(cfg)
	expected := cfg.Aisles * cfg.BaysPerAisle * cfg.Levels * (cfg.Depths + 1)
	if len(layout.AllCells()) != expected {
		t.Fatalf("expected %d cells", expected)
	}
}

func TestInBounds(t *testing.T) {
	layout := config.NewLayout(config.DefaultWarehouse())
	if !layout.InBounds(layout.CraneHome("01")) {
		t.Fatal("home should be in bounds")
	}
}

func TestLoadFromEnv(t *testing.T) {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Warehouse.Name == "" {
		t.Fatal("warehouse name required")
	}
}
