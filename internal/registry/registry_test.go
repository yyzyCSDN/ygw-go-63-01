package registry

import (
	"testing"

	"modelrouter/internal/metric"
	"modelrouter/internal/model"
)

func TestRegisterAndInstances(t *testing.T) {
	reg := New(&StaticProvider{}, metric.NullRecorder{})
	v, err := reg.Register("resnet", "v1")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if v.State != model.StateRegistered {
		t.Fatalf("state = %v, want registered", v.State)
	}
	if _, err := reg.Register("resnet", "v1"); err == nil {
		t.Fatal("duplicate register must fail")
	}
	if err := reg.AddInstance("resnet", "v1", "a", "10.0.0.1:9000"); err != nil {
		t.Fatalf("add instance: %v", err)
	}
	if err := reg.AddInstance("missing", "v1", "a", "10.0.0.1:9000"); err == nil {
		t.Fatal("add instance to unknown version must fail")
	}
	ver, ok := reg.LookupVersion("resnet", "v1")
	if !ok || len(ver.Instances) != 1 {
		t.Fatalf("lookup version failed: ok=%v instances=%d", ok, len(ver.Instances))
	}
}

func TestRetireAndActiveVersions(t *testing.T) {
	reg := New(&StaticProvider{}, metric.NullRecorder{})
	_, _ = reg.Register("resnet", "v1")
	_ = reg.AddInstance("resnet", "v1", "a", "10.0.0.1:9000")
	if err := reg.Retire("resnet", "v1"); err != nil {
		t.Fatalf("retire: %v", err)
	}
	states := reg.VersionStates("resnet")
	if states["v1"] != "retired" {
		t.Fatalf("state = %v, want retired", states["v1"])
	}
	active := reg.ActiveVersions()
	if _, ok := active["resnet"]; ok {
		t.Fatal("retired version must not stay active")
	}
}

func TestSyncAppliesSummary(t *testing.T) {
	summary := model.SyncSummary{
		Models: []string{"resnet"},
		Active: map[string]string{"resnet": "v2"},
	}
	reg := New(&StaticProvider{Summary: summary}, metric.NullRecorder{})
	if err := reg.Sync(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if reg.ActiveVersion("resnet") != "v2" {
		t.Fatalf("active version = %q, want v2", reg.ActiveVersion("resnet"))
	}
	models := reg.Models()
	if len(models) != 1 || models[0] != "resnet" {
		t.Fatalf("models = %v", models)
	}
	if reg.Generation() == 0 {
		t.Fatal("generation must advance after sync")
	}
	last := reg.LastSync()
	if last.At.IsZero() {
		t.Fatal("last sync time must be recorded")
	}
}

func TestAliasMapDirect(t *testing.T) {
	reg := New(&StaticProvider{}, metric.NullRecorder{})
	_, _ = reg.Register("resnet", "v1")
	_, _ = reg.Register("resnet", "v2")
	if err := reg.SetAlias("stable", "resnet", "v2"); err != nil {
		t.Fatalf("set alias: %v", err)
	}
	target, ok := reg.ResolveAlias("stable")
	if !ok || target.Version != "v2" {
		t.Fatalf("alias target = %+v ok=%v", target, ok)
	}
	aliases := reg.AliasesSnapshot()
	if aliases["stable"].Version != "v2" {
		t.Fatalf("alias snapshot = %+v", aliases)
	}
	names := reg.AliasNames()
	if len(names) != 1 || names[0] != "stable" {
		t.Fatalf("alias names = %v", names)
	}
}

func TestSnapshotStates(t *testing.T) {
	reg := New(&StaticProvider{}, metric.NullRecorder{})
	_, _ = reg.Register("resnet", "v1")
	snapshot := reg.Snapshot()
	if snapshot["resnet/v1"] != "registered" {
		t.Fatalf("snapshot = %v", snapshot)
	}
}
