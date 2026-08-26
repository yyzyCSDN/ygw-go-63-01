package model

// InstanceState describes the current health state of a serving instance.
type InstanceState int

const (
	StateUnknown InstanceState = iota
	StateHealthy
	StateDraining
	StateRemoved
)

// String returns a stable name for the instance state.
func (s InstanceState) String() string {
	switch s {
	case StateUnknown:
		return "unknown"
	case StateHealthy:
		return "healthy"
	case StateDraining:
		return "draining"
	case StateRemoved:
		return "removed"
	}
	return "unknown"
}

// Instance is one replica that can serve a model version.
type Instance struct {
	ID      string
	Model   string
	Version string
	Addr    string
	State   InstanceState
}

// NewInstance creates an instance record in the unknown state.
func NewInstance(model, version, id, addr string) *Instance {
	return &Instance{
		ID:      id,
		Model:   model,
		Version: version,
		Addr:    addr,
		State:   StateUnknown,
	}
}

// Healthy reports whether the instance can accept traffic.
func (i *Instance) Healthy() bool {
	return i.State == StateHealthy
}

// SetState updates the instance state and returns the previous state.
func (i *Instance) SetState(next InstanceState) InstanceState {
	prev := i.State
	i.State = next
	return prev
}

// Key returns a stable identifier for the instance.
func (i *Instance) Key() string {
	return KeyFor(i.Model, i.Version, i.ID)
}

// KeyFor builds a stable instance key from its parts.
func KeyFor(modelName, version, id string) string {
	return modelName + "/" + version + "/" + id
}
