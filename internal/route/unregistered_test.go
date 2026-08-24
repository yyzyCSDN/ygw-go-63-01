package route

import (
	"errors"
	"testing"

	"modelrouter/internal/health"
	"modelrouter/internal/metric"
	"modelrouter/internal/model"
	"modelrouter/internal/registry"
)

// TestResolveUnregisteredModel asserts that routing an unregistered model
// name returns ErrModelNotFound instead of crashing the gateway.
//
// Regression guard: registry.Lookup used to return (nil, true) for a model
// with no active version, so Resolve skipped its not-found guard and
// dereferenced a nil version record, panicking the request goroutine.
func TestResolveUnregisteredModel(t *testing.T) {
	reg := registry.New(&registry.StaticProvider{}, metric.NullRecorder{})
	ht := health.NewTracker(metric.NullRecorder{})
	table := NewTable(reg, ht, NewGrayManager(100), metric.NullRecorder{})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Resolve panicked on unregistered model: %v", r)
		}
	}()

	_, err := table.Resolve("definitely-not-registered", "req-1")
	if err == nil {
		t.Fatal("expected error for unregistered model, got nil")
	}
	if !errors.Is(err, model.ErrModelNotFound) {
		t.Fatalf("expected ErrModelNotFound, got %v", err)
	}
}
