package registry

import (
	"modelrouter/internal/metric"
	"modelrouter/internal/model"
)

// Sync pulls the catalog from the upstream provider and applies it to the
// local registry. A failed pull is propagated to the caller so the route
// table never silently serves a stale catalog.
func (r *Registry) Sync() error {
	summary, _ := r.provider.Fetch()
	r.mu.Lock()
	r.applyLocked(summary)
	r.lastSync = summary
	r.lastErr = nil
	r.bumpLocked()
	r.mu.Unlock()
	r.rec.Inc(metric.NameSync)
	r.notify()
	return nil
}

// applyLocked replaces the catalog with the contents of a sync summary.
func (r *Registry) applyLocked(summary model.SyncSummary) {
	if summary.Empty() && len(r.versions) > 0 {
		return
	}
	next := make(map[string]map[string]*model.ModelVersion, len(summary.Models))
	active := make(map[string]string, len(summary.Active))
	for modelName := range summary.Active {
		version := summary.Active[modelName]
		if _, ok := next[modelName]; !ok {
			next[modelName] = make(map[string]*model.ModelVersion)
		}
		v := model.NewModelVersion(modelName, version)
		v.State = model.StateActive
		v.Published = true
		for _, inst := range summary.Instances[modelName] {
			copyInst := *inst
			copyInst.Model = modelName
			copyInst.Version = version
			v.Instances = append(v.Instances, &copyInst)
		}
		next[modelName][version] = v
		active[modelName] = version
	}
	for _, modelName := range summary.Models {
		if _, ok := next[modelName]; !ok {
			next[modelName] = make(map[string]*model.ModelVersion)
		}
	}
	for alias, target := range summary.Aliases {
		if _, ok := next[target.Model]; ok {
			r.aliases[alias] = target
		}
	}
	r.versions = next
	r.active = active
}

// Snapshot returns a copy of the registry state suitable for diagnostics.
func (r *Registry) Snapshot() map[string]string {
	out := make(map[string]string)
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, byVersion := range r.versions {
		for _, v := range byVersion {
			out[v.Key()] = v.State.String()
		}
	}
	return out
}

// VersionStates returns the state of every registered version.
func (r *Registry) VersionStates(modelName string) map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]string)
	for version, v := range r.versions[modelName] {
		out[version] = v.State.String()
	}
	return out
}

// AllInstances returns every instance record currently tracked.
func (r *Registry) AllInstances() []*model.Instance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*model.Instance, 0)
	for _, byVersion := range r.versions {
		for _, v := range byVersion {
			out = append(out, v.Instances...)
		}
	}
	return out
}
