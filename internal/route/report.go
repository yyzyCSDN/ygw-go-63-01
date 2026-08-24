package route

import (
	"fmt"
	"sort"

	"modelrouter/internal/model"
)

// TargetReport describes the targets cached for one model.
type TargetReport struct {
	InstanceID string `json:"instance_id"`
	Addr       string `json:"addr"`
	Version    string `json:"version"`
}

// ModelRouteReport describes the routing state of one model.
type ModelRouteReport struct {
	Model      string          `json:"model"`
	Version    string          `json:"version"`
	GrayRatio  float64         `json:"gray_ratio"`
	Sequence   int             `json:"sequence"`
	Targets    []TargetReport  `json:"targets"`
	ChunkSizes []int           `json:"chunk_sizes"`
}

// RouteReport is a diagnostic snapshot of the route table.
type RouteReport struct {
	Models []ModelRouteReport `json:"models"`
}

// Report builds a diagnostic snapshot of the route table.
func (t *Table) Report() RouteReport {
	t.mu.RLock()
	active := make(map[string]string, len(t.active))
	for name, version := range t.active {
		active[name] = version
	}
	targets := make(map[string][]TargetReport, len(t.targets))
	for name, list := range t.targets {
		reports := make([]TargetReport, 0, len(list))
		for _, target := range list {
			reports = append(reports, TargetReport{
				InstanceID: target.InstanceID,
				Addr:       target.Addr,
				Version:    target.Version,
			})
		}
		targets[name] = reports
	}
	t.mu.RUnlock()
	names := make([]string, 0, len(active))
	for name := range active {
		names = append(names, name)
	}
	sort.Strings(names)
	report := RouteReport{}
	for _, name := range names {
		version := active[name]
		reports := targets[name]
		chunkSize := t.s.ChunkSize(len(reports))
		views := make([]model.Target, 0, len(reports))
		for _, report := range reports {
			views = append(views, model.Target{
				Model:      name,
				Version:    report.Version,
				InstanceID: report.InstanceID,
				Addr:       report.Addr,
			})
		}
		chunks := chunkTargets(views, chunkSize)
		sizes := make([]int, 0, len(chunks))
		for _, chunk := range chunks {
			sizes = append(sizes, len(chunk))
		}
		report.Models = append(report.Models, ModelRouteReport{
			Model:      name,
			Version:    version,
			GrayRatio:  t.gray.Ratio(name),
			Sequence:   t.gray.Sequence(name),
			Targets:    reports,
			ChunkSizes: sizes,
		})
	}
	return report
}

// Describe returns a one-line summary of the route table.
func (t *Table) Describe() string {
	report := t.Report()
	return fmt.Sprintf("route table with %d models", len(report.Models))
}
