package route

import (
	"fmt"
	"sync"

	"modelrouter/internal/health"
	"modelrouter/internal/metric"
	"modelrouter/internal/model"
	"modelrouter/internal/registry"
)

// Table resolves requests to concrete targets. It reads the live registry
// snapshot for the active version and keeps a derived target cache that is
// rebuilt whenever the registry notifies it about a change.
type Table struct {
	reg    *registry.Registry
	health *health.Tracker
	gray   *GrayManager
	rec    metric.Recorder
	s      *Selector

	mu       sync.RWMutex
	active   map[string]string
	aliases  map[string]model.AliasTarget
	aliasVer map[string]string
	targets  map[string][]model.Target
}

// NewTable builds a route table and populates it from the current registry
// snapshot.
func NewTable(reg *registry.Registry, ht *health.Tracker, gm *GrayManager, rec metric.Recorder) *Table {
	t := &Table{
		reg:    reg,
		health: ht,
		gray:   gm,
		rec:    rec,
		s:      NewSelector(rec),
	}
	t.RefreshFromRegistry()
	return t
}

// RefreshFromRegistry copies the registry snapshot into the local caches and
// rebuilds the derived target lists.
func (t *Table) RefreshFromRegistry() {
	active := t.reg.ActiveVersions()
	aliases := t.reg.AliasesSnapshot()
	t.mu.Lock()
	t.active = active
	t.aliases = aliases
	t.aliasVer = make(map[string]string, len(aliases))
	for alias, target := range aliases {
		t.aliasVer[alias] = target.Version
	}
	t.targets = make(map[string][]model.Target, len(active))
	for modelName, version := range active {
		t.targets[modelName] = t.buildTargets(modelName, version)
	}
	t.mu.Unlock()
}

// Resolve picks the target for a model request.
func (t *Table) Resolve(modelName, requestID string) (model.RouteResult, error) {
	ver, ok := t.reg.Lookup(modelName)
	if !ok {
		return model.RouteResult{}, model.ErrModelNotFound
	}
	return t.resolveVersion(ver, requestID)
}

// ResolveAlias resolves an alias to a concrete route result.
func (t *Table) ResolveAlias(alias, requestID string) (model.RouteResult, error) {
	t.mu.RLock()
	target, ok := t.aliases[alias]
	version := t.aliasVer[alias]
	t.mu.RUnlock()
	if !ok {
		return model.RouteResult{}, model.ErrAliasNotFound
	}
	if version == "" {
		version = target.Version
	}
	ver, ok := t.reg.LookupVersion(target.Model, version)
	if !ok || ver.State != model.StateActive {
		return model.RouteResult{}, model.ErrVersionNotFound
	}
	return t.resolveVersion(ver, requestID)
}

// resolveVersion builds a route result for an already active version record.
func (t *Table) resolveVersion(ver *model.ModelVersion, requestID string) (model.RouteResult, error) {
	if ver == nil {
		return model.RouteResult{}, model.ErrModelNotFound
	}
	targets := t.targetsFor(ver.Model, ver)
	if len(targets) == 0 {
		return model.RouteResult{}, model.ErrTargetUnavailable
	}
	chunks := chunkTargets(targets, t.s.ChunkSize(len(targets)))
	selected := t.s.Pick(chunks, ver.Model, requestID)
	result := model.RouteResult{
		Model:   ver.Model,
		Version: ver.Version,
		Target:  selected,
		Gray:    t.gray.Decide(ver.Model),
	}
	result.Fallbacks = t.fallbacksFor(ver.Model, ver.Version, selected.InstanceID)
	t.rec.Inc(metric.NameRoute)
	return result, nil
}

// targetsFor returns the cached targets for a version, rebuilding them when
// the cache is empty or points at a different version.
func (t *Table) targetsFor(modelName string, ver *model.ModelVersion) []model.Target {
	t.mu.RLock()
	cached := t.targets[modelName]
	t.mu.RUnlock()
	if len(cached) == 0 || cached[0].Version != ver.Version {
		return t.buildTargets(modelName, ver.Version)
	}
	return cached
}

// buildTargets converts the healthy instances of a version into targets.
func (t *Table) buildTargets(modelName, version string) []model.Target {
	ver, ok := t.reg.LookupVersion(modelName, version)
	if !ok {
		return nil
	}
	healthy := make([]*model.Instance, 0, len(ver.Instances))
	for _, inst := range ver.Instances {
		if t.health.IsHealthy(inst.Model, inst.Version, inst.ID) {
			healthy = append(healthy, inst)
		}
	}
	if len(healthy) == 0 {
		healthy = ver.HealthyInstances()
	}
	out := make([]model.Target, 0, len(healthy))
	for _, inst := range healthy {
		out = append(out, ver.TargetFor(inst, false))
	}
	return out
}

// fallbacksFor returns the other healthy targets of the same version.
func (t *Table) fallbacksFor(modelName, version, excludeID string) []model.Target {
	t.mu.RLock()
	targets := t.targets[modelName]
	t.mu.RUnlock()
	out := make([]model.Target, 0, len(targets))
	for _, target := range targets {
		if target.InstanceID != excludeID {
			out = append(out, target)
		}
	}
	return out
}

// Snapshot returns a diagnostic view of the cached routing state.
func (t *Table) Snapshot() map[string]string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make(map[string]string, len(t.active))
	for modelName, version := range t.active {
		out[modelName] = fmt.Sprintf("%s (%d targets)", version, len(t.targets[modelName]))
	}
	return out
}

// Decide advances the per-model request sequence and reports whether the
// request falls inside the configured gray window. The sequence resets to
// zero at every period boundary so consecutive windows never overlap.
func (g *GrayManager) Decide(modelName string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	window, ok := g.windows[modelName]
	if !ok {
		return false
	}
	seq := g.seq[modelName]
	total := g.period
	g.seq[modelName] = seq + 1
	if g.seq[modelName] >= total {
		// Start a fresh, non-overlapping window at position 0 so the gray
		// segment [0, gray) is never carried across the boundary.
		g.seq[modelName] = 0
	}
	return window.Contains(seq, total)
}

// Sequence returns the current sequence position of a model.
func (g *GrayManager) Sequence(modelName string) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.seq[modelName]
}
