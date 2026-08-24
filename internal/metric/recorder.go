package metric

// Recorder is the narrow interface components use to emit metrics without
// depending on the full Metrics type.
type Recorder interface {
	Inc(name string)
	Add(name string, delta int64)
	Set(name string, value int64)
}

// NullRecorder discards every metric event and is used by tests and by the
// static provider paths that do not need telemetry.
type NullRecorder struct{}

// Inc implements Recorder.
func (NullRecorder) Inc(string) {}

// Add implements Recorder.
func (NullRecorder) Add(string, int64) {}

// Set implements Recorder.
func (NullRecorder) Set(string, int64) {}

// MetricsRecorder adapts *Metrics to the Recorder interface.
type MetricsRecorder struct {
	metrics *Metrics
}

// NewRecorder wraps a Metrics instance as a Recorder.
func NewRecorder(metrics *Metrics) *MetricsRecorder {
	return &MetricsRecorder{metrics: metrics}
}

// Inc implements Recorder.
func (r *MetricsRecorder) Inc(name string) {
	r.metrics.Counter(name).Inc()
}

// Add implements Recorder.
func (r *MetricsRecorder) Add(name string, delta int64) {
	r.metrics.Counter(name).Add(delta)
}

// Set implements Recorder.
func (r *MetricsRecorder) Set(name string, value int64) {
	r.metrics.Gauge(name).Set(value)
}
