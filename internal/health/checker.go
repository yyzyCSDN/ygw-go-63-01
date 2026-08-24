package health

import (
	"context"
	"time"

	"modelrouter/internal/metric"
	"modelrouter/internal/model"
)

// ProbeFunc reports whether an instance address answers a health probe.
type ProbeFunc func(addr string) bool

// Checker periodically probes the known instances and feeds the results
// into the tracker.
type Checker struct {
	tracker  *Tracker
	probe    ProbeFunc
	interval time.Duration
	rec      metric.Recorder
}

// NewChecker creates a checker with the given probe function.
func NewChecker(tracker *Tracker, probe ProbeFunc, interval time.Duration, rec metric.Recorder) *Checker {
	return &Checker{
		tracker:  tracker,
		probe:    probe,
		interval: interval,
		rec:      rec,
	}
}

// Run executes the probe loop until the context is cancelled.
func (c *Checker) Run(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	c.CheckOnce()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.CheckOnce()
		}
	}
}

// CheckOnce probes every healthy-or-unknown instance once and returns the
// number of instances that became healthy.
func (c *Checker) CheckOnce() int {
	healthy := 0
	for _, inst := range c.tracker.HealthyList() {
		if c.probe(inst.Addr) {
			healthy++
		} else {
			_ = c.tracker.MarkDraining(inst.Model, inst.Version, inst.ID)
		}
	}
	c.rec.Set(metric.NameHealthObserved, int64(c.tracker.Size()))
	return healthy
}

// ProbeAll marks every tracked instance according to the probe result.
func (c *Checker) ProbeAll(instances []*model.Instance) int {
	healthy := 0
	for _, inst := range instances {
		if c.probe(inst.Addr) {
			err := c.tracker.MarkHealthy(inst.Model, inst.Version, inst.ID)
			if err != nil {
				if observeErr := c.tracker.Observe(inst); observeErr != nil {
					continue
				}
				if err = c.tracker.MarkHealthy(inst.Model, inst.Version, inst.ID); err != nil {
					continue
				}
			}
			healthy++
		}
	}
	return healthy
}
