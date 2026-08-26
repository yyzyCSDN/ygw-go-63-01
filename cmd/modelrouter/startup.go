package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"modelrouter/internal/dispatch"
	"modelrouter/internal/fallback"
	"modelrouter/internal/gateway"
	"modelrouter/internal/health"
	"modelrouter/internal/metric"
	"modelrouter/internal/model"
	"modelrouter/internal/quota"
	"modelrouter/internal/registry"
	"modelrouter/internal/route"
)

// seedCatalog is the optional startup catalog payload.
type seedCatalog struct {
	Models []seedModel `json:"models"`
}

type seedModel struct {
	Name      string        `json:"name"`
	Version   string        `json:"version"`
	Instances []seedInstance `json:"instances"`
}

type seedInstance struct {
	ID   string `json:"id"`
	Addr string `json:"addr"`
}

// BuildComponents wires the full gateway and returns the HTTP server.
func BuildComponents(cfg *Config) (*Server, error) {
	metrics := metric.New()
	rec := metric.NewRecorder(metrics)
	provider, err := buildProvider(cfg)
	if err != nil {
		return nil, err
	}
	reg := registry.New(provider, rec)
	healthTracker := health.NewTracker(rec)
	gray := route.NewGrayManager(cfg.GrayPeriod)
	table := route.NewTable(reg, healthTracker, gray, rec)
	reg.SetOnChange(table.RefreshFromRegistry)
	if err := reg.Sync(); err != nil {
		return nil, fmt.Errorf("seed sync: %w", err)
	}
	table.RefreshFromRegistry()

	checker := health.NewChecker(healthTracker, probeInstance, cfg.HealthCheck, rec)
	checker.ProbeAll(reg.AllInstances())
	limiter := quota.New(cfg.Quota)
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: cfg.Quota,
			MaxConnsPerHost:     cfg.Quota,
		},
	}
	transport := dispatch.NewTransport(client, rec)
	dispatcher := dispatch.New(transport, cfg.Timeout, rec)
	executor, err := fallback.New(fallback.DefaultPolicy(), rec)
	if err != nil {
		return nil, err
	}
	gw := gateway.New(reg, table, limiter, dispatcher, executor, rec)
	return NewServer(cfg, gw, metrics, healthTracker, checker, reg), nil
}

func buildProvider(cfg *Config) (registry.UpstreamProvider, error) {
	if cfg.Seed == "" {
		return &registry.StaticProvider{}, nil
	}
	var catalog seedCatalog
	if err := json.Unmarshal([]byte(cfg.Seed), &catalog); err != nil {
		return nil, fmt.Errorf("parse MODELROUTER_SEED: %w", err)
	}
	summary := model.SyncSummary{
		Active:    make(map[string]string),
		Aliases:   make(map[string]model.AliasTarget),
		Instances: make(map[string][]*model.Instance),
		At:        time.Now().UTC(),
	}
	for _, seed := range catalog.Models {
		summary.Models = append(summary.Models, seed.Name)
		summary.Active[seed.Name] = seed.Version
		for _, instance := range seed.Instances {
			inst := model.NewInstance(seed.Name, seed.Version, instance.ID, instance.Addr)
			inst.SetState(model.StateHealthy)
			summary.Instances[seed.Name] = append(summary.Instances[seed.Name], inst)
		}
	}
	summary.Count = len(summary.Models)
	return &registry.StaticProvider{Summary: summary}, nil
}

func probeInstance(addr string) bool {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get("http://" + addr + "/v1/healthz")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
