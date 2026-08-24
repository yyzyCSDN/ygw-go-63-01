package quota

import (
	"testing"

	"modelrouter/internal/model"
)

func TestAcquireAndRelease(t *testing.T) {
	limiter := New(4)
	slots := make([]int, 0, 4)
	for i := 0; i < 4; i++ {
		slot, err := limiter.Acquire()
		if err != nil {
			t.Fatalf("acquire %d: %v", i, err)
		}
		slots = append(slots, slot)
	}
	if limiter.InFlight() != 4 {
		t.Fatalf("in-flight = %d, want 4", limiter.InFlight())
	}
	if _, err := limiter.Acquire(); err != model.ErrQuotaExceeded {
		t.Fatalf("full limiter must reject, got %v", err)
	}
	for _, slot := range slots {
		if err := limiter.Release(slot); err != nil {
			t.Fatalf("release %d: %v", slot, err)
		}
	}
	if limiter.InFlight() != 0 {
		t.Fatalf("in-flight = %d, want 0", limiter.InFlight())
	}
}

func TestReleaseInvalidSlot(t *testing.T) {
	limiter := New(2)
	if err := limiter.Release(-1); err == nil {
		t.Fatal("negative slot must be rejected")
	}
	if err := limiter.Release(5); err == nil {
		t.Fatal("out-of-range slot must be rejected")
	}
}

func TestCapacity(t *testing.T) {
	limiter := New(7)
	if limiter.Capacity() != 7 {
		t.Fatalf("capacity = %d", limiter.Capacity())
	}
	if limiter.CounterState() != 0 {
		t.Fatalf("counter = %d, want 0", limiter.CounterState())
	}
}

func TestNormalOperationBelowBoundary(t *testing.T) {
	limiter := New(2)
	for round := 0; round < 40; round++ {
		a, err := limiter.Acquire()
		if err != nil {
			t.Fatalf("round %d acquire: %v", round, err)
		}
		b, err := limiter.Acquire()
		if err != nil {
			t.Fatalf("round %d acquire: %v", round, err)
		}
		_ = limiter.Release(a)
		_ = limiter.Release(b)
	}
}
