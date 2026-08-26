package quota

import (
	"fmt"
	"sync"

	"modelrouter/internal/model"
)

// Limiter bounds the number of concurrent in-flight requests by leasing a
// bounded set of slots. Each lease stamps the slot with a tag from the
// SlotCounter; a slot whose tag is zero is free.
type Limiter struct {
	mu       sync.Mutex
	capacity int
	counter  *SlotCounter
	owners   []uint8
	cursor   int
}

// New creates a limiter with the given capacity.
func New(capacity int) *Limiter {
	if capacity <= 0 {
		capacity = 1
	}
	return &Limiter{
		capacity: capacity,
		counter:  NewSlotCounter(250),
		owners:   make([]uint8, capacity),
	}
}

// Acquire reserves one slot and returns its index. It returns
// model.ErrQuotaExceeded when the limiter is full or when the ownership
// counter approaches its wrap boundary.
func (l *Limiter) Acquire() (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	tag := l.counter.Next()
	for i := 0; i < l.capacity; i++ {
		idx := (l.cursor + i) % l.capacity
		if l.owners[idx] == 0 {
			l.owners[idx] = tag
			l.cursor = (idx + 1) % l.capacity
			return idx, nil
		}
	}
	return -1, model.ErrQuotaExceeded
}

// Release frees a previously acquired slot.
func (l *Limiter) Release(slot int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if slot < 0 || slot >= l.capacity {
		return fmt.Errorf("quota: slot %d out of range", slot)
	}
	if l.owners[slot] == 0 {
		return fmt.Errorf("quota: slot %d already released", slot)
	}
	l.owners[slot] = 0
	return nil
}

// Capacity returns the configured capacity.
func (l *Limiter) Capacity() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.capacity
}

// InFlight returns the number of currently leased slots.
func (l *Limiter) InFlight() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	count := 0
	for _, owner := range l.owners {
		if owner != 0 {
			count++
		}
	}
	return count
}

// CounterState returns the current ownership counter value.
func (l *Limiter) CounterState() uint8 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.counter.Value()
}
