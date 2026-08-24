package metric

import (
	"sort"
	"sync"
	"sync/atomic"
)

// Counter is a monotonically increasing integer metric.
type Counter struct {
	name  string
	value atomic.Int64
}

// Inc adds one to the counter.
func (c *Counter) Inc() {
	c.value.Add(1)
}

// Add adds n to the counter.
func (c *Counter) Add(n int64) {
	c.value.Add(n)
}

// Value returns the current counter value.
func (c *Counter) Value() int64 {
	return c.value.Load()
}

// Gauge is a metric that can go up and down.
type Gauge struct {
	name  string
	value atomic.Int64
}

// Set stores the current gauge value.
func (g *Gauge) Set(n int64) {
	g.value.Store(n)
}

// Add changes the gauge by n.
func (g *Gauge) Add(n int64) {
	g.value.Add(n)
}

// Value returns the current gauge value.
func (g *Gauge) Value() int64 {
	return g.value.Load()
}

// Metrics holds named counters and gauges for the whole gateway process.
type Metrics struct {
	mu       sync.RWMutex
	counters map[string]*Counter
	gauges   map[string]*Gauge
}

// New creates an empty metric registry.
func New() *Metrics {
	return &Metrics{
		counters: make(map[string]*Counter),
		gauges:   make(map[string]*Gauge),
	}
}

// Counter returns (creating if needed) the named counter.
func (m *Metrics) Counter(name string) *Counter {
	m.mu.RLock()
	c, ok := m.counters[name]
	m.mu.RUnlock()
	if ok {
		return c
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok = m.counters[name]; ok {
		return c
	}
	c = &Counter{name: name}
	m.counters[name] = c
	return c
}

// Gauge returns (creating if needed) the named gauge.
func (m *Metrics) Gauge(name string) *Gauge {
	m.mu.RLock()
	g, ok := m.gauges[name]
	m.mu.RUnlock()
	if ok {
		return g
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if g, ok = m.gauges[name]; ok {
		return g
	}
	g = &Gauge{name: name}
	m.gauges[name] = g
	return g
}

// Snapshot returns a sorted copy of all counter values.
func (m *Metrics) Snapshot() map[string]int64 {
	out := make(map[string]int64)
	m.mu.RLock()
	defer m.mu.RUnlock()
	for name, c := range m.counters {
		out[name] = c.Value()
	}
	for name, g := range m.gauges {
		out[name] = g.Value()
	}
	return out
}

// Names returns all registered metric names sorted.
func (m *Metrics) Names() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	seen := make(map[string]struct{}, len(m.counters)+len(m.gauges))
	for name := range m.counters {
		seen[name] = struct{}{}
	}
	for name := range m.gauges {
		seen[name] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
