package registry

import (
	"time"

	"modelrouter/internal/model"
)

// UpstreamProvider supplies model catalog snapshots for synchronization.
type UpstreamProvider interface {
	Fetch() (model.SyncSummary, error)
}

// StaticProvider returns a fixed summary and is used by the gateway when it
// boots with a seed catalog.
type StaticProvider struct {
	Summary model.SyncSummary
	Err     error
}

// Fetch implements UpstreamProvider.
func (p *StaticProvider) Fetch() (model.SyncSummary, error) {
	if p.Err != nil {
		return model.SyncSummary{}, p.Err
	}
	summary := p.Summary
	if summary.At.IsZero() {
		summary.At = time.Now().UTC()
	}
	return summary, nil
}

// ErrorProvider is an upstream that always fails; used in tests and fault
// injection.
type ErrorProvider struct {
	Err error
}

// Fetch implements UpstreamProvider.
func (p *ErrorProvider) Fetch() (model.SyncSummary, error) {
	return model.SyncSummary{}, p.Err
}
