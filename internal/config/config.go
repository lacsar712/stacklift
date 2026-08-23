package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/lacsar712/stacklift/internal/model"
)

type Config struct {
	Warehouse WarehouseConfig
	Limits    model.LimitSet
	ClockStep int64
	DemoMode  bool
}

type WarehouseConfig struct {
	Name         string
	Aisles       int
	BaysPerAisle int
	Levels       int
	Depths       int
	BayPitchMM   int64
	LevelPitchMM int64
	DepthPitchMM int64
}

func Default() Config {
	return Config{Warehouse: DefaultWarehouse(), Limits: model.DefaultLimits(), ClockStep: 100}
}

func DefaultWarehouse() WarehouseConfig {
	return WarehouseConfig{
		Name: "WH-01", Aisles: 4, BaysPerAisle: 20, Levels: 8, Depths: 2,
		BayPitchMM: 2800, LevelPitchMM: 1800, DepthPitchMM: 1200,
	}
}

func (c Config) Validate() error {
	if err := c.Limits.Validate(); err != nil {
		return fmt.Errorf("config limits: %w", err)
	}
	if err := c.Warehouse.Validate(); err != nil {
		return fmt.Errorf("config warehouse: %w", err)
	}
	if c.ClockStep <= 0 {
		return fmt.Errorf("config: clock_step must be positive")
	}
	return nil
}

func (w WarehouseConfig) Validate() error {
	if w.Aisles <= 0 {
		return fmt.Errorf("warehouse: aisles must be positive")
	}
	if w.BaysPerAisle <= 0 || w.Levels <= 0 {
		return fmt.Errorf("warehouse: bays and levels must be positive")
	}
	if w.BayPitchMM <= 0 || w.LevelPitchMM <= 0 {
		return fmt.Errorf("warehouse: pitch values must be positive")
	}
	return nil
}

func (w WarehouseConfig) TotalCells() int {
	return w.Aisles * w.BaysPerAisle * w.Levels * (w.Depths + 1)
}

func LoadFromEnv() (Config, error) {
	cfg := Default()
	if v := os.Getenv("STACKLIFT_DEMO"); v == "1" || v == "true" {
		cfg.DemoMode = true
	}
	if v := os.Getenv("STACKLIFT_CLOCK_STEP_MS"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return cfg, fmt.Errorf("STACKLIFT_CLOCK_STEP_MS: %w", err)
		}
		cfg.ClockStep = n
	}
	if v := os.Getenv("STACKLIFT_WAREHOUSE"); v != "" {
		cfg.Warehouse.Name = v
	}
	return cfg, cfg.Validate()
}
