package quota

import (
	"errors"
	"testing"

	"modelrouter/internal/model"
)

func TestQuotaCounterNoOverflow(t *testing.T) {
	limiter := New(2)
	limiter.counter.value = 254
	slot, err := limiter.Acquire()
	if err == nil {
		_ = limiter.Release(slot)
		t.Fatal("quota must reject when the ownership counter reaches the wrap boundary")
	}
	if !errors.Is(err, model.ErrQuotaExceeded) {
		t.Fatalf("unexpected error at boundary: %v", err)
	}
}
