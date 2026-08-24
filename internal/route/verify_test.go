package route

import (
	"errors"
	"testing"

	"modelrouter/internal/health"
	"modelrouter/internal/metric"
	"modelrouter/internal/model"
	"modelrouter/internal/registry"
)

func TestUnregisteredModelNoNilPanic(t *testing.T) {
	reg := registry.New(&registry.StaticProvider{}, metric.NullRecorder{})
	ht := health.NewTracker(metric.NullRecorder{})
	table := NewTable(reg, ht, NewGrayManager(100), metric.NullRecorder{})
	_, err := table.Resolve("not-registered", "req-1")
	if err == nil {
		t.Fatal("unregistered model must return an error, not panic")
	}
	if !errors.Is(err, model.ErrModelNotFound) {
		t.Fatalf("unexpected error for unregistered model: %v", err)
	}
}
