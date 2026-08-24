package registry_test

import (
	"testing"

	"modelrouter/internal/metric"
	"modelrouter/internal/registry"
)

// TestLookupUnregisteredModelIsNotFound pins the contract that used to be
// broken: a model with no active version must report ok=false so callers
// surface model-not-found instead of dereferencing a nil record.
//
// Before the fix Lookup returned (nil, true), which let route.Resolve skip
// its not-found guard and panic on the nil pointer.
func TestLookupUnregisteredModelIsNotFound(t *testing.T) {
	reg := registry.New(&registry.StaticProvider{}, metric.NullRecorder{})

	ver, ok := reg.Lookup("definitely-not-registered")
	if ok {
		t.Fatalf("Lookup reported found for unregistered model: ver=%v ok=%v", ver, ok)
	}
	if ver != nil {
		t.Fatalf("Lookup must return nil record when not found, got %v", ver)
	}
}
