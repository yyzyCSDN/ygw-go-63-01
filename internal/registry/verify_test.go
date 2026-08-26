package registry

import (
	"errors"
	"testing"

	"modelrouter/internal/metric"
	"modelrouter/internal/model"
)

func TestRegistrySyncErrorNotSwallowed(t *testing.T) {
	providerErr := errors.New("upstream unavailable")
	reg := New(&ErrorProvider{Err: providerErr}, metric.NullRecorder{})
	err := reg.Sync()
	if err == nil {
		t.Fatal("registry sync failure must be surfaced, not swallowed")
	}
	if !errors.Is(err, model.ErrSyncFailed) {
		t.Fatalf("unexpected error: %v", err)
	}
}
