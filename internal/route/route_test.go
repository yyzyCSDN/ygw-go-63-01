package route

import (
	"testing"

	"modelrouter/internal/health"
	"modelrouter/internal/metric"
	"modelrouter/internal/model"
	"modelrouter/internal/registry"
)

func newTestTable(t *testing.T, modelName, version string, instanceCount int) (*Table, *registry.Registry) {
	t.Helper()
	instances := make([]*model.Instance, 0, instanceCount)
	for i := 0; i < instanceCount; i++ {
		inst := model.NewInstance(modelName, version, string(rune('a'+i)), "10.0.0.1:9000")
		inst.SetState(model.StateHealthy)
		instances = append(instances, inst)
	}
	summary := model.SyncSummary{
		Models:    []string{modelName},
		Active:    map[string]string{modelName: version},
		Instances: map[string][]*model.Instance{modelName: instances},
	}
	reg := registry.New(&registry.StaticProvider{Summary: summary}, metric.NullRecorder{})
	if err := reg.Sync(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	ht := health.NewTracker(metric.NullRecorder{})
	table := NewTable(reg, ht, NewGrayManager(100), metric.NullRecorder{})
	return table, reg
}

func TestResolveRegisteredModel(t *testing.T) {
	table, _ := newTestTable(t, "resnet", "v1", 4)
	result, err := table.Resolve("resnet", "req-1")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if result.Version != "v1" || result.Target.Addr == "" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Fallbacks) < 1 {
		t.Fatalf("fallbacks must be available, got %d", len(result.Fallbacks))
	}
}

func TestResolveIsDeterministic(t *testing.T) {
	table, _ := newTestTable(t, "resnet", "v1", 4)
	first, err := table.Resolve("resnet", "req-1")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	second, err := table.Resolve("resnet", "req-1")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if first.Target.InstanceID != second.Target.InstanceID {
		t.Fatalf("same request must pick the same target: %s vs %s", first.Target.InstanceID, second.Target.InstanceID)
	}
}

func TestChunkTargetsExactMultiple(t *testing.T) {
	targets := []model.Target{
		{InstanceID: "a"}, {InstanceID: "b"}, {InstanceID: "c"}, {InstanceID: "d"},
	}
	chunks := chunkTargets(targets, 2)
	if len(chunks) != 2 {
		t.Fatalf("chunks = %d, want 2", len(chunks))
	}
	count := 0
	for _, chunk := range chunks {
		count += len(chunk)
	}
	if count != 4 {
		t.Fatalf("chunked count = %d, want 4", count)
	}
}

func TestGrayManagerBasics(t *testing.T) {
	gray := NewGrayManager(100)
	if err := gray.SetWindow("resnet", 0.5); err != nil {
		t.Fatalf("set window: %v", err)
	}
	if gray.Ratio("resnet") != 0.5 {
		t.Fatalf("ratio = %v", gray.Ratio("resnet"))
	}
	if err := gray.SetWindow("resnet", 1.5); err == nil {
		t.Fatal("ratio above 1 must be rejected")
	}
	if gray.Sequence("resnet") != 0 {
		t.Fatal("sequence must start at zero")
	}
	window := GrayWindow{Ratio: 1}
	grayCount, baseCount := window.Split(100)
	if grayCount != 100 || baseCount != 0 {
		t.Fatalf("full window split = %d/%d", grayCount, baseCount)
	}
	zero := GrayWindow{Ratio: 0}
	grayCount, baseCount = zero.Split(100)
	if grayCount != 0 || baseCount != 100 {
		t.Fatalf("empty window split = %d/%d", grayCount, baseCount)
	}
}

func TestTableSnapshot(t *testing.T) {
	table, _ := newTestTable(t, "resnet", "v1", 2)
	snapshot := table.Snapshot()
	if snapshot["resnet"] == "" {
		t.Fatalf("snapshot missing model: %v", snapshot)
	}
}

func TestSelectorPicksValidTarget(t *testing.T) {
	selector := NewSelector(metric.NullRecorder{})
	targets := []model.Target{
		{InstanceID: "a"}, {InstanceID: "b"}, {InstanceID: "c"}, {InstanceID: "d"},
	}
	chunks := chunkTargets(targets, selector.ChunkSize(len(targets)))
	picked := selector.Pick(chunks, "resnet", "req-1")
	if picked.InstanceID == "" {
		t.Fatal("selector returned an empty target")
	}
}

// TestResolveAliasReflectsRepoint reproduces the cached-alias symptom: once
// the registry repoints the "stable" alias at a new version, the next
// ResolveAlias call must route to the new version immediately, without any
// stale v1 traffic leaking through the alias.
func TestResolveAliasReflectsRepoint(t *testing.T) {
	const (
		modelName = "resnet"
		v1, v2    = "v1", "v2"
	)
	// Two versions, each with one healthy instance at a distinct address so
	// the selected target's version tells us which mapping was used.
	summary := model.SyncSummary{
		Models: []string{modelName},
		Active: map[string]string{modelName: v1},
		Instances: map[string][]*model.Instance{modelName: {
			model.NewInstance(modelName, v1, "a", "10.0.0.1:9000"),
		}},
	}
	reg := registry.New(&registry.StaticProvider{Summary: summary}, metric.NullRecorder{})
	if err := reg.Sync(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	// Register and publish v2 so the alias can point at it.
	if _, err := reg.Register(modelName, v2); err != nil {
		t.Fatalf("register v2: %v", err)
	}
	if err := reg.AddInstance(modelName, v2, "b", "10.0.0.2:9000"); err != nil {
		t.Fatalf("add instance v2: %v", err)
	}
	if err := reg.Publish(modelName, v2); err != nil {
		t.Fatalf("publish v2: %v", err)
	}

	ht := health.NewTracker(metric.NullRecorder{})
	// Both versions' instances must be healthy for the selector to pick a
	// target; observe them in the tracker the way the gateway does.
	for _, inst := range reg.AllInstances() {
		healthy := model.NewInstance(inst.Model, inst.Version, inst.ID, inst.Addr)
		healthy.SetState(model.StateHealthy)
		_ = ht.Observe(healthy)
	}
	table := NewTable(reg, ht, NewGrayManager(100), metric.NullRecorder{})
	// Wire the change listener the same way the gateway does, so the route
	// table refreshes its cached alias mapping whenever the registry mutates.
	reg.SetOnChange(table.RefreshFromRegistry)

	// Point stable at v1 first.
	if err := reg.SetAlias("stable", modelName, v1); err != nil {
		t.Fatalf("set alias v1: %v", err)
	}
	first, err := table.ResolveAlias("stable", "req-1")
	if err != nil {
		t.Fatalf("resolve alias v1: %v", err)
	}
	if first.Version != v1 {
		t.Fatalf("first resolve = %s, want %s", first.Version, v1)
	}

	// Ops repoints stable to v2. The very next resolution must use v2 — no
	// stale v1 traffic may reach stable.
	if err := reg.SetAlias("stable", modelName, v2); err != nil {
		t.Fatalf("set alias v2: %v", err)
	}
	second, err := table.ResolveAlias("stable", "req-1")
	if err != nil {
		t.Fatalf("resolve alias v2: %v", err)
	}
	if second.Version != v2 {
		t.Fatalf("after repoint, resolve = %s, want %s (alias cache not invalidated)", second.Version, v2)
	}
	if second.Target.Version != v2 {
		t.Fatalf("after repoint, target version = %s, want %s", second.Target.Version, v2)
	}
	// Sanity: the registry's own alias target must also reflect v2.
	target, ok := reg.ResolveAlias("stable")
	if !ok || target.Version != v2 {
		t.Fatalf("registry alias target = %+v ok=%v, want v2", target, ok)
	}
}
