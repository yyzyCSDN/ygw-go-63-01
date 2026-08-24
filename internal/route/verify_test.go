package route

import (
	"testing"

	"modelrouter/internal/health"
	"modelrouter/internal/metric"
	"modelrouter/internal/model"
	"modelrouter/internal/registry"
)

func TestAliasResolutionUsesLatestMapping(t *testing.T) {
	reg := registry.New(&registry.StaticProvider{}, metric.NullRecorder{})
	ht := health.NewTracker(metric.NullRecorder{})
	table := NewTable(reg, ht, NewGrayManager(100), metric.NullRecorder{})
	reg.SetOnChange(table.RefreshFromRegistry)

	if _, err := reg.Register("resnet", "v1"); err != nil {
		t.Fatalf("register v1: %v", err)
	}
	if err := reg.AddInstance("resnet", "v1", "a", "10.0.0.1:9001"); err != nil {
		t.Fatalf("add instance: %v", err)
	}
	if err := ht.Observe(model.NewInstance("resnet", "v1", "a", "10.0.0.1:9001")); err != nil {
		t.Fatalf("observe instance: %v", err)
	}
	if err := ht.MarkHealthy("resnet", "v1", "a"); err != nil {
		t.Fatalf("mark healthy: %v", err)
	}
	if err := reg.Publish("resnet", "v1"); err != nil {
		t.Fatalf("publish v1: %v", err)
	}
	if err := reg.SetAlias("stable", "resnet", "v1"); err != nil {
		t.Fatalf("set alias v1: %v", err)
	}
	table.RefreshFromRegistry()

	first, err := table.ResolveAlias("stable", "req-1")
	if err != nil {
		t.Fatalf("initial alias resolution: %v", err)
	}
	if first.Version != "v1" {
		t.Fatalf("initial alias version = %s, want v1", first.Version)
	}

	if _, err := reg.Register("resnet", "v2"); err != nil {
		t.Fatalf("register v2: %v", err)
	}
	if err := reg.AddInstance("resnet", "v2", "b", "10.0.0.2:9002"); err != nil {
		t.Fatalf("add instance v2: %v", err)
	}
	if err := ht.Observe(model.NewInstance("resnet", "v2", "b", "10.0.0.2:9002")); err != nil {
		t.Fatalf("observe instance v2: %v", err)
	}
	if err := ht.MarkHealthy("resnet", "v2", "b"); err != nil {
		t.Fatalf("mark healthy v2: %v", err)
	}
	if err := reg.Publish("resnet", "v2"); err != nil {
		t.Fatalf("publish v2: %v", err)
	}
	if err := reg.SetAlias("stable", "resnet", "v2"); err != nil {
		t.Fatalf("set alias v2: %v", err)
	}

	second, err := table.ResolveAlias("stable", "req-2")
	if err != nil {
		t.Fatalf("alias resolution after switch: %v", err)
	}
	if second.Version != "v2" {
		t.Fatalf("alias must use the latest mapping after switch, still got %s", second.Version)
	}
}
