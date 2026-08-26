package health

import (
	"fmt"
	"sync"

	"modelrouter/internal/metric"
	"modelrouter/internal/model"
)

// Tracker keeps the observed health state of every known instance and
// produces the snapshots the route table consumes.
type Tracker struct {
	mu        sync.RWMutex
	instances map[string]*model.Instance
	rec       metric.Recorder
}

// NewTracker creates an empty health tracker.
func NewTracker(rec metric.Recorder) *Tracker {
	return &Tracker{
		instances: make(map[string]*model.Instance),
		rec:       rec,
	}
}

// Observe stores a new instance record or updates the state of an existing
// one.
func (t *Tracker) Observe(inst *model.Instance) error {
	if inst == nil {
		return fmt.Errorf("health: nil instance")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	key := inst.Key()
	if prev, ok := t.instances[key]; ok {
		prev.SetState(inst.State)
		t.rec.Inc(metric.NameHealthObserved)
		return nil
	}
	t.instances[key] = inst
	t.rec.Inc(metric.NameHealthObserved)
	return nil
}

// MarkHealthy transitions an instance into the healthy state.
func (t *Tracker) MarkHealthy(modelName, version, id string) error {
	return t.transition(modelName, version, id, model.StateHealthy)
}

// MarkDraining transitions an instance into the draining state.
func (t *Tracker) MarkDraining(modelName, version, id string) error {
	return t.transition(modelName, version, id, model.StateDraining)
}

// MarkRemoved removes an instance from the tracker.
func (t *Tracker) MarkRemoved(modelName, version, id string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.instances, model.KeyFor(modelName, version, id))
	t.rec.Inc(metric.NameHealthObserved)
	return nil
}

func (t *Tracker) transition(modelName, version, id string, state model.InstanceState) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	inst, ok := t.instances[model.KeyFor(modelName, version, id)]
	if !ok {
		return fmt.Errorf("health: unknown instance %s", id)
	}
	inst.SetState(state)
	t.rec.Inc(metric.NameHealthObserved)
	return nil
}

// HealthyList returns the current healthy instances in stable order.
func (t *Tracker) HealthyList() []*model.Instance {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]*model.Instance, 0, len(t.instances))
	for _, inst := range t.instances {
		if inst.Healthy() {
			out = append(out, cloneInstance(inst))
		}
	}
	sortInstances(out)
	return out
}

// IsHealthy reports whether the tracker currently considers an instance
// healthy.
func (t *Tracker) IsHealthy(modelName, version, id string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	inst, ok := t.instances[model.KeyFor(modelName, version, id)]
	return ok && inst.Healthy()
}

// Snapshot returns an immutable snapshot of the healthy instances.
func (t *Tracker) Snapshot() Snapshot {
	return Snapshot{instances: t.HealthyList()}
}

// Size reports how many instances are tracked in total.
func (t *Tracker) Size() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.instances)
}

func cloneInstance(inst *model.Instance) *model.Instance {
	copyInst := *inst
	return &copyInst
}

func sortInstances(instances []*model.Instance) {
	for i := 1; i < len(instances); i++ {
		for j := i; j > 0 && instances[j].Key() < instances[j-1].Key(); j-- {
			instances[j], instances[j-1] = instances[j-1], instances[j]
		}
	}
}
