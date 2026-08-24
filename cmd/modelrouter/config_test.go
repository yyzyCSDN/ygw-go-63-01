package main

import (
	"testing"
	"time"
)

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("MODELROUTER_ADDR", "")
	t.Setenv("MODELROUTER_TIMEOUT_MS", "")
	t.Setenv("MODELROUTER_QUOTA", "")
	t.Setenv("MODELROUTER_GRAY_PERIOD", "")
	t.Setenv("MODELROUTER_HEALTH_MS", "")
	t.Setenv("MODELROUTER_SEED", "")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Addr != ":8080" || cfg.Timeout != 5*time.Second || cfg.Quota != 64 || cfg.GrayPeriod != 100 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("MODELROUTER_ADDR", "127.0.0.1:9999")
	t.Setenv("MODELROUTER_TIMEOUT_MS", "250")
	t.Setenv("MODELROUTER_QUOTA", "16")
	t.Setenv("MODELROUTER_GRAY_PERIOD", "50")
	t.Setenv("MODELROUTER_HEALTH_MS", "2000")
	t.Setenv("MODELROUTER_SEED", "{}")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Addr != "127.0.0.1:9999" || cfg.Timeout != 250*time.Millisecond || cfg.Quota != 16 || cfg.GrayPeriod != 50 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.HealthCheck != 2*time.Second || cfg.Seed != "{}" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadConfigRejectsBadValues(t *testing.T) {
	t.Setenv("MODELROUTER_QUOTA", "0")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("zero quota must be rejected")
	}
	t.Setenv("MODELROUTER_QUOTA", "64")
	t.Setenv("MODELROUTER_SEED", "{not-json")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("bad seed must be rejected by config load")
	}
}

func TestRequestIDGenerator(t *testing.T) {
	first := newRequestID()
	second := newRequestID()
	if first == "" || first == second {
		t.Fatalf("request ids must be non-empty and distinct: %q %q", first, second)
	}
}
