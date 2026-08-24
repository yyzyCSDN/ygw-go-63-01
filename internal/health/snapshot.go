package health

import "modelrouter/internal/model"

// Snapshot is an immutable view of the healthy instances at one point in
// time.
type Snapshot struct {
	instances []*model.Instance
}

// Instances returns the healthy instances in the snapshot.
func (s Snapshot) Instances() []*model.Instance {
	return s.instances
}

// Count returns the number of healthy instances in the snapshot.
func (s Snapshot) Count() int {
	return len(s.instances)
}

// Empty reports whether the snapshot carries no healthy instances.
func (s Snapshot) Empty() bool {
	return len(s.instances) == 0
}

// IDs returns the instance ids in the snapshot.
func (s Snapshot) IDs() []string {
	out := make([]string, 0, len(s.instances))
	for _, inst := range s.instances {
		out = append(out, inst.ID)
	}
	return out
}
