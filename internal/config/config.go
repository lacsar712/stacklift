package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/lacsar712/stacklift/internal/model"
)

type Config struct {
	ListenAddr      string            `json:"listen_addr"`
	ProcessTickMS   int64             `json:"process_tick_ms"`
	WindHoldWindow  int64             `json:"wind_hold_window_ticks"`
	RigIDs          []string          `json:"rig_ids"`
	Limits          model.LimitSet    `json:"limits"`
	Interlock       InterlockConfig   `json:"interlock"`
	JournalPath     string            `json:"journal_path"`
	TelemetryBuffer int               `json:"telemetry_buffer"`
	Meta            map[string]string `json:"meta"`
}

type InterlockConfig struct {
	SlewBoomMutex   bool    `json:"slew_boom_mutex"`
	MomentDeratePct float64 `json:"moment_derate_pct"`
	HoldAfterFaultS int     `json:"hold_after_fault_s"`
}

func Default() Config {
	return Config{
		ListenAddr: ":8092", ProcessTickMS: 100, WindHoldWindow: 30,
		RigIDs: []string{"TC-01", "TC-02"}, Limits: model.DefaultLimits(),
		Interlock: InterlockConfig{SlewBoomMutex: true, MomentDeratePct: 75, HoldAfterFaultS: 5},
		TelemetryBuffer: 512,
		Meta: map[string]string{"service": "stacklift", "domain": "tower-crane-moment-wind-interlock"},
	}
}

func LoadFile(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("config read: %w", err)
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("config json: %w", err)
	}
	return cfg, cfg.Validate()
}

func FromEnv(cfg Config) Config {
	if v := os.Getenv("CRANESAFE_LISTEN"); v != "" {
		cfg.ListenAddr = v
	}
	if v := os.Getenv("CRANESAFE_TICK_MS"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			cfg.ProcessTickMS = n
		}
	}
	if v := os.Getenv("CRANESAFE_WIND_HOLD"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			cfg.WindHoldWindow = n
		}
	}
	return cfg
}

func (c Config) Validate() error {
	if c.ListenAddr == "" {
		return fmt.Errorf("listen_addr required")
	}
	if c.ProcessTickMS <= 0 || c.WindHoldWindow <= 0 || len(c.RigIDs) == 0 || c.TelemetryBuffer <= 0 {
		return fmt.Errorf("invalid config")
	}
	return c.Limits.Validate()
}

func (c Config) TickDuration() time.Duration {
	return time.Duration(c.ProcessTickMS) * time.Millisecond
}

func (c Config) WindHoldDuration() time.Duration {
	return time.Duration(c.WindHoldWindow*c.ProcessTickMS) * time.Millisecond
}

func (c Config) RigKnown(id string) bool {
	for _, r := range c.RigIDs {
		if r == id {
			return true
		}
	}
	return false
}
