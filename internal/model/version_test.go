package model

import "testing"

func TestVersionLifecycleHelpers(t *testing.T) {
	v := NewModelVersion("resnet", "v1")
	if v.State != StateDraft {
		t.Fatalf("new version must start as draft, got %v", v.State)
	}
	if v.CanPublish() {
		t.Fatal("draft version must not be publishable")
	}
	if v.Key() != "resnet/v1" {
		t.Fatalf("unexpected key: %s", v.Key())
	}
	v.State = StateRegistered
	if !v.CanPublish() {
		t.Fatal("registered version must be publishable")
	}
}

func TestVersionHealthyInstances(t *testing.T) {
	v := NewModelVersion("resnet", "v1")
	v.Instances = append(v.Instances,
		NewInstance("resnet", "v1", "a", "10.0.0.1:9000"),
		NewInstance("resnet", "v1", "b", "10.0.0.2:9000"),
	)
	v.Instances[0].State = StateHealthy
	healthy := v.HealthyInstances()
	if len(healthy) != 1 || healthy[0].ID != "a" {
		t.Fatalf("unexpected healthy list: %+v", healthy)
	}
}

func TestTargetForAndResponse(t *testing.T) {
	v := NewModelVersion("resnet", "v1")
	v.State = StateActive
	inst := NewInstance("resnet", "v1", "a", "10.0.0.1:9000")
	inst.State = StateHealthy
	target := v.TargetFor(inst, true)
	if !target.Gray || target.Addr != "10.0.0.1:9000" || target.Version != "v1" {
		t.Fatalf("unexpected target: %+v", target)
	}
	if !(Response{Status: 204, Target: target}).Success() {
		t.Fatal("2xx response must be successful")
	}
	if (Response{Status: 503, Target: target}).Success() {
		t.Fatal("5xx response must not be successful")
	}
}

func TestInstanceKeyAndState(t *testing.T) {
	inst := NewInstance("resnet", "v1", "a", "10.0.0.1:9000")
	if inst.Key() != "resnet/v1/a" {
		t.Fatalf("unexpected instance key: %s", inst.Key())
	}
	if inst.Healthy() {
		t.Fatal("new instance must not be healthy")
	}
	prev := inst.SetState(StateHealthy)
	if prev != StateUnknown || !inst.Healthy() {
		t.Fatalf("unexpected state transition: %v -> %v", prev, inst.State)
	}
}

func TestRouteResultCandidates(t *testing.T) {
	result := RouteResult{
		Model:   "resnet",
		Version: "v1",
		Target:  Target{Model: "resnet", Version: "v1", InstanceID: "a"},
		Fallbacks: []Target{
			{Model: "resnet", Version: "v1", InstanceID: "b"},
		},
	}
	candidates := result.Candidates()
	if len(candidates) != 2 || candidates[0].InstanceID != "a" || candidates[1].InstanceID != "b" {
		t.Fatalf("unexpected candidates: %+v", candidates)
	}
}

func TestSyncSummaryEmpty(t *testing.T) {
	if !(SyncSummary{}).Empty() {
		t.Fatal("empty summary must report empty")
	}
	if (SyncSummary{Models: []string{"resnet"}}).Empty() {
		t.Fatal("summary with models must not report empty")
	}
}
