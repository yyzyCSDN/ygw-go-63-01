package model

// VersionState describes the lifecycle of a model version inside the
// registry. A version moves from draft through registered to active and can
// eventually be retired when it is no longer routable.
type VersionState int

const (
	StateDraft VersionState = iota
	StateRegistered
	StateActive
	StateRetired
)

// String returns a stable name for the version state.
func (s VersionState) String() string {
	switch s {
	case StateDraft:
		return "draft"
	case StateRegistered:
		return "registered"
	case StateActive:
		return "active"
	case StateRetired:
		return "retired"
	}
	return "unknown"
}

// ModelVersion is one version of a deployed model together with the
// instances that can serve it.
type ModelVersion struct {
	Model     string
	Version   string
	State     VersionState
	Published bool
	Instances []*Instance
	GrayRatio float64
}

// NewModelVersion creates a version record in the draft state.
func NewModelVersion(model, version string) *ModelVersion {
	return &ModelVersion{
		Model:   model,
		Version: version,
		State:   StateDraft,
	}
}

// CanPublish reports whether the version is allowed to become active.
func (v *ModelVersion) CanPublish() bool {
	return v.State == StateRegistered || v.State == StateActive
}

// HealthyInstances returns only the instances that are currently healthy.
func (v *ModelVersion) HealthyInstances() []*Instance {
	out := make([]*Instance, 0, len(v.Instances))
	for _, inst := range v.Instances {
		if inst.State == StateHealthy {
			out = append(out, inst)
		}
	}
	return out
}

// TargetFor converts a healthy instance into a routable target.
func (v *ModelVersion) TargetFor(inst *Instance, gray bool) Target {
	return Target{
		Model:      v.Model,
		Version:    v.Version,
		InstanceID: inst.ID,
		Addr:       inst.Addr,
		Gray:       gray,
	}
}

// Key returns a stable identifier for the version.
func (v *ModelVersion) Key() string {
	return v.Model + "/" + v.Version
}
