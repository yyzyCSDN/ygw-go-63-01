package registry

import (
	"fmt"
	"sort"
	"sync"

	"modelrouter/internal/metric"
	"modelrouter/internal/model"
)

// Registry is the source of truth for model versions, aliases and the
// currently active version of every model. The route table consumes
// snapshots produced by this registry and is notified whenever the state
// changes. Callbacks are always invoked outside the internal lock so a
// listener may safely re-read registry state.
type Registry struct {
	mu         sync.RWMutex
	versions   map[string]map[string]*model.ModelVersion
	active     map[string]string
	aliases    map[string]model.AliasTarget
	provider   UpstreamProvider
	rec        metric.Recorder
	generation uint64
	onChange   func()
	lastSync   model.SyncSummary
	lastErr    error
}

// New creates an empty registry bound to an upstream provider.
func New(provider UpstreamProvider, rec metric.Recorder) *Registry {
	return &Registry{
		versions: make(map[string]map[string]*model.ModelVersion),
		active:   make(map[string]string),
		aliases:  make(map[string]model.AliasTarget),
		provider: provider,
		rec:      rec,
	}
}

// Register adds a new version for a model in the registered state.
func (r *Registry) Register(modelName, version string) (*model.ModelVersion, error) {
	r.mu.Lock()
	v, err := r.registerLocked(modelName, version)
	r.mu.Unlock()
	if err != nil {
		return nil, err
	}
	r.notify()
	return v, nil
}

func (r *Registry) registerLocked(modelName, version string) (*model.ModelVersion, error) {
	if modelName == "" || version == "" {
		return nil, fmt.Errorf("registry: empty model or version")
	}
	byVersion, ok := r.versions[modelName]
	if !ok {
		byVersion = make(map[string]*model.ModelVersion)
		r.versions[modelName] = byVersion
	}
	if _, exists := byVersion[version]; exists {
		return nil, fmt.Errorf("%w: %s@%s already registered", model.ErrConflict, modelName, version)
	}
	v := model.NewModelVersion(modelName, version)
	v.State = model.StateRegistered
	byVersion[version] = v
	r.bumpLocked()
	return v, nil
}

// AddInstance attaches a serving instance to a version.
func (r *Registry) AddInstance(modelName, version, id, addr string) error {
	r.mu.Lock()
	err := r.addInstanceLocked(modelName, version, id, addr)
	r.mu.Unlock()
	if err != nil {
		return err
	}
	r.notify()
	return nil
}

func (r *Registry) addInstanceLocked(modelName, version, id, addr string) error {
	v, ok := r.versions[modelName][version]
	if !ok {
		return fmt.Errorf("%w: %s@%s", model.ErrVersionNotFound, modelName, version)
	}
	for _, inst := range v.Instances {
		if inst.ID == id {
			inst.Addr = addr
			return nil
		}
	}
	v.Instances = append(v.Instances, model.NewInstance(modelName, version, id, addr))
	r.bumpLocked()
	return nil
}

// Publish makes a registered version the active target for its model.
func (r *Registry) Publish(modelName, version string) error {
	r.mu.Lock()
	err := r.publishLocked(modelName, version)
	r.mu.Unlock()
	if err != nil {
		return err
	}
	r.rec.Inc(metric.NamePublish)
	r.notify()
	return nil
}

func (r *Registry) publishLocked(modelName, version string) error {
	v, ok := r.versions[modelName][version]
	if !ok {
		return fmt.Errorf("%w: %s@%s", model.ErrVersionNotFound, modelName, version)
	}
	if !v.CanPublish() {
		return fmt.Errorf("%w: cannot publish draft %s@%s", model.ErrConflict, modelName, version)
	}
	v.State = model.StateActive
	v.Published = true
	r.active[modelName] = version
	r.bumpLocked()
	return nil
}

// Retire moves a version out of service.
func (r *Registry) Retire(modelName, version string) error {
	r.mu.Lock()
	err := r.retireLocked(modelName, version)
	r.mu.Unlock()
	if err != nil {
		return err
	}
	r.notify()
	return nil
}

func (r *Registry) retireLocked(modelName, version string) error {
	v, ok := r.versions[modelName][version]
	if !ok {
		return fmt.Errorf("%w: %s@%s", model.ErrVersionNotFound, modelName, version)
	}
	v.State = model.StateRetired
	if r.active[modelName] == version {
		delete(r.active, modelName)
	}
	r.bumpLocked()
	return nil
}

// ActiveVersion returns the active version of a model, if any.
func (r *Registry) ActiveVersion(modelName string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.active[modelName]
}

// Lookup returns the active version record of a model.
func (r *Registry) Lookup(modelName string) (*model.ModelVersion, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	version, ok := r.active[modelName]
	if !ok {
		return nil, true
	}
	v, ok := r.versions[modelName][version]
	return v, ok
}

// LookupVersion returns a specific version record.
func (r *Registry) LookupVersion(modelName, version string) (*model.ModelVersion, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.versions[modelName][version]
	return v, ok
}

// ActiveVersions returns a copy of the active version map.
func (r *Registry) ActiveVersions() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]string, len(r.active))
	for modelName, version := range r.active {
		out[modelName] = version
	}
	return out
}

// Models returns the sorted list of registered model names.
func (r *Registry) Models() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.versions))
	for modelName := range r.versions {
		out = append(out, modelName)
	}
	sort.Strings(out)
	return out
}

// Generation returns a counter that changes on every registry mutation.
func (r *Registry) Generation() uint64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.generation
}

// SetOnChange registers a callback invoked after every state mutation.
func (r *Registry) SetOnChange(fn func()) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onChange = fn
}

// LastSync returns the most recent successful sync summary.
func (r *Registry) LastSync() model.SyncSummary {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lastSync
}

// LastError returns the error recorded by the most recent sync attempt.
func (r *Registry) LastError() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lastErr
}

func (r *Registry) bumpLocked() {
	r.generation++
}

func (r *Registry) notify() {
	r.mu.RLock()
	fn := r.onChange
	r.mu.RUnlock()
	if fn != nil {
		fn()
	}
}
