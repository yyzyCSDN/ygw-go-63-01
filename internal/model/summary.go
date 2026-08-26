package model

import "time"

// SyncSummary is the payload returned by an upstream registry provider.
type SyncSummary struct {
	Models  []string
	Active  map[string]string // model -> active version
	Aliases map[string]AliasTarget
	Instances map[string][]*Instance // model -> instance records
	Count   int
	At      time.Time
}

// Empty reports whether the summary carries no model data.
func (s SyncSummary) Empty() bool {
	return len(s.Models) == 0 && len(s.Active) == 0
}
