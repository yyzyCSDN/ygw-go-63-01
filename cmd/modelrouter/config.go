package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds every tunable of the gateway process.
type Config struct {
	Addr        string
	Timeout     time.Duration
	Quota       int
	GrayPeriod  int
	HealthCheck time.Duration
	Seed        string
}

// DefaultConfig returns the production defaults.
func DefaultConfig() Config {
	return Config{
		Addr:        ":8080",
		Timeout:     5 * time.Second,
		Quota:       64,
		GrayPeriod:  100,
		HealthCheck: 10 * time.Second,
	}
}

// LoadConfig reads configuration from the environment.
func LoadConfig() (*Config, error) {
	cfg := DefaultConfig()
	if value := os.Getenv("MODELROUTER_ADDR"); value != "" {
		cfg.Addr = value
	}
	if value := os.Getenv("MODELROUTER_TIMEOUT_MS"); value != "" {
		ms, err := strconv.Atoi(value)
		if err != nil || ms <= 0 {
			return nil, fmt.Errorf("invalid MODELROUTER_TIMEOUT_MS %q", value)
		}
		cfg.Timeout = time.Duration(ms) * time.Millisecond
	}
	if value := os.Getenv("MODELROUTER_QUOTA"); value != "" {
		quota, err := strconv.Atoi(value)
		if err != nil || quota <= 0 {
			return nil, fmt.Errorf("invalid MODELROUTER_QUOTA %q", value)
		}
		cfg.Quota = quota
	}
	if value := os.Getenv("MODELROUTER_GRAY_PERIOD"); value != "" {
		period, err := strconv.Atoi(value)
		if err != nil || period <= 0 {
			return nil, fmt.Errorf("invalid MODELROUTER_GRAY_PERIOD %q", value)
		}
		cfg.GrayPeriod = period
	}
	if value := os.Getenv("MODELROUTER_HEALTH_MS"); value != "" {
		ms, err := strconv.Atoi(value)
		if err != nil || ms <= 0 {
			return nil, fmt.Errorf("invalid MODELROUTER_HEALTH_MS %q", value)
		}
		cfg.HealthCheck = time.Duration(ms) * time.Millisecond
	}
	cfg.Seed = os.Getenv("MODELROUTER_SEED")
	if cfg.Seed != "" && !json.Valid([]byte(cfg.Seed)) {
		return nil, fmt.Errorf("MODELROUTER_SEED must be valid JSON")
	}
	return &cfg, nil
}
