package registry

import (
	"fmt"

	"modelrouter/internal/metric"
	"modelrouter/internal/model"
)

// SetAlias points an alias such as "stable" at a concrete model version.
func (r *Registry) SetAlias(alias, modelName, version string) error {
	r.mu.Lock()
	err := func() error {
		if _, ok := r.versions[modelName][version]; !ok {
			return fmt.Errorf("%w: %s@%s", model.ErrVersionNotFound, modelName, version)
		}
		r.aliases[alias] = model.AliasTarget{Model: modelName, Version: version}
		r.bumpLocked()
		return nil
	}()
	r.mu.Unlock()
	if err != nil {
		return err
	}
	r.rec.Inc(metric.NameAlias)
	r.notify()
	return nil
}

// ResolveAlias returns the current target of an alias.
func (r *Registry) ResolveAlias(alias string) (model.AliasTarget, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	target, ok := r.aliases[alias]
	return target, ok
}

// AliasesSnapshot returns a copy of the alias map.
func (r *Registry) AliasesSnapshot() map[string]model.AliasTarget {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]model.AliasTarget, len(r.aliases))
	for alias, target := range r.aliases {
		out[alias] = target
	}
	return out
}

// AliasNames returns the sorted list of registered aliases.
func (r *Registry) AliasNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.aliases))
	for alias := range r.aliases {
		out = append(out, alias)
	}
	sortStrings(out)
	return out
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
