package health

import (
	"testing"

	"modelrouter/internal/metric"
	"modelrouter/internal/model"
)

func TestTrackerObserveAndTransition(t *testing.T) {
	tracker := NewTracker(metric.NullRecorder{})
	inst := model.NewInstance("resnet", "v1", "a", "10.0.0.1:9000")
	if err := tracker.Observe(inst); err != nil {
		t.Fatalf("observe: %v", err)
	}
	if err := tracker.MarkHealthy("resnet", "v1", "a"); err != nil {
		t.Fatalf("mark healthy: %v", err)
	}
	if err := tracker.MarkDraining("resnet", "v1", "a"); err != nil {
		t.Fatalf("mark draining: %v", err)
	}
	if tracker.Size() != 1 {
		t.Fatalf("size = %d, want 1", tracker.Size())
	}
	if err := tracker.MarkRemoved("resnet", "v1", "a"); err != nil {
		t.Fatalf("mark removed: %v", err)
	}
	if tracker.Size() != 0 {
		t.Fatalf("size = %d, want 0", tracker.Size())
	}
}

func TestSnapshotListsHealthyInstances(t *testing.T) {
	tracker := NewTracker(metric.NullRecorder{})
	for _, id := range []string{"a", "b", "c"} {
		inst := model.NewInstance("resnet", "v1", id, "10.0.0.1:9000")
		if err := tracker.Observe(inst); err != nil {
			t.Fatalf("observe %s: %v", id, err)
		}
		if err := tracker.MarkHealthy("resnet", "v1", id); err != nil {
			t.Fatalf("mark healthy %s: %v", id, err)
		}
	}
	snapshot := tracker.Snapshot()
	if snapshot.Empty() {
		t.Fatal("snapshot must not be empty")
	}
	instances := snapshot.Instances()
	if len(instances) != 3 {
		t.Fatalf("healthy instances = %d, want 3", len(instances))
	}
	ids := snapshot.IDs()
	if len(ids) != 3 || ids[0] != "a" || ids[2] != "c" {
		t.Fatalf("unexpected ids: %v", ids)
	}
}

func TestCheckerProbeAll(t *testing.T) {
	tracker := NewTracker(metric.NullRecorder{})
	checker := NewChecker(tracker, func(string) bool { return true }, 0, metric.NullRecorder{})
	instances := []*model.Instance{
		model.NewInstance("resnet", "v1", "a", "10.0.0.1:9000"),
		model.NewInstance("resnet", "v1", "b", "10.0.0.2:9000"),
	}
	healthy := checker.ProbeAll(instances)
	if healthy != 2 {
		t.Fatalf("healthy = %d, want 2", healthy)
	}
	if len(tracker.Snapshot().Instances()) != 2 {
		t.Fatalf("snapshot instances = %d, want 2", len(tracker.Snapshot().Instances()))
	}
}
