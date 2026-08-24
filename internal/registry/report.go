package registry

import (
	"sort"
	"time"

	"modelrouter/internal/model"
)

// VersionReport describes one registered version for diagnostics.
type VersionReport struct {
	Version   string `json:"version"`
	State     string `json:"state"`
	Published bool   `json:"published"`
	Instances int    `json:"instances"`
}

// ModelReport describes one model for diagnostics.
type ModelReport struct {
	Model      string                   `json:"model"`
	Active     string                   `json:"active"`
	Versions   []VersionReport          `json:"versions"`
	Aliases    []model.AliasTarget      `json:"aliases"`
}

// RegistryReport is a point-in-time diagnostic view of the registry.
type RegistryReport struct {
	Models     []ModelReport    `json:"models"`
	Generation uint64           `json:"generation"`
	LastSyncAt time.Time        `json:"last_sync_at"`
	LastError  string           `json:"last_error"`
}

// Report builds a diagnostic snapshot of the registry.
func (r *Registry) Report() RegistryReport {
	r.mu.RLock()
	defer r.mu.RUnlock()
	report := RegistryReport{
		Generation: r.generation,
		LastSyncAt: r.lastSync.At,
	}
	if r.lastErr != nil {
		report.LastError = r.lastErr.Error()
	}
	modelNames := make([]string, 0, len(r.versions))
	for name := range r.versions {
		modelNames = append(modelNames, name)
	}
	sort.Strings(modelNames)
	for _, name := range modelNames {
		modelReport := ModelReport{
			Model:  name,
			Active: r.active[name],
		}
		versionNames := make([]string, 0, len(r.versions[name]))
		for version := range r.versions[name] {
			versionNames = append(versionNames, version)
		}
		sort.Strings(versionNames)
		for _, version := range versionNames {
			v := r.versions[name][version]
			modelReport.Versions = append(modelReport.Versions, VersionReport{
				Version:   version,
				State:     v.State.String(),
				Published: v.Published,
				Instances: len(v.Instances),
			})
		}
		aliasNames := make([]string, 0, len(r.aliases))
		for alias := range r.aliases {
			aliasNames = append(aliasNames, alias)
		}
		sort.Strings(aliasNames)
		for _, alias := range aliasNames {
			if r.aliases[alias].Model == name {
				modelReport.Aliases = append(modelReport.Aliases, r.aliases[alias])
			}
		}
		report.Models = append(report.Models, modelReport)
	}
	return report
}
