package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Config struct {
	Addr                  string  `json:"addr"`
	DataDir               string  `json:"data_dir"`
	BackwashTimeoutMS     int     `json:"backwash_timeout_ms"`
	FlowTimeoutMS         int     `json:"flow_timeout_ms"`
	TurbidityHigh         float64 `json:"turbidity_high"`
	ChlorineLow           float64 `json:"chlorine_low"`
	MaxRecords            int     `json:"max_records"`
	DosingIntervalSec     int     `json:"dosing_interval_sec"`
	RotateIntervalSec     int     `json:"rotate_interval_sec"`
	MaxConcurrentBackwash int     `json:"max_concurrent_backwash"`
	PressureThreshold     float64 `json:"pressure_threshold"`
}

func Default() *Config {
	return &Config{
		Addr:                  "127.0.0.1:18080",
		DataDir:               "runtime",
		BackwashTimeoutMS:     20000,
		FlowTimeoutMS:         15000,
		TurbidityHigh:         1.0,
		ChlorineLow:           0.3,
		MaxRecords:            100000,
		DosingIntervalSec:     5,
		RotateIntervalSec:     120,
		MaxConcurrentBackwash: 2,
		PressureThreshold:     0.25,
	}
}

func Load(path string) (*Config, error) {
	if path == "" {
		return Default(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	cfg := Default()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

func (c *Config) BackwashTimeout() time.Duration {
	return time.Duration(c.BackwashTimeoutMS) * time.Millisecond
}

func (c *Config) FlowTimeout() time.Duration {
	return time.Duration(c.FlowTimeoutMS) * time.Millisecond
}

func (c *Config) DosingInterval() time.Duration {
	return time.Duration(c.DosingIntervalSec) * time.Second
}

func (c *Config) RotateInterval() time.Duration {
	return time.Duration(c.RotateIntervalSec) * time.Second
}
