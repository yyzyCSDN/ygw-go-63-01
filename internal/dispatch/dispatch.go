package dispatch

import (
	"context"
	"sync/atomic"
	"time"

	"modelrouter/internal/metric"
	"modelrouter/internal/model"
)

// Dispatcher forwards an inference request to a concrete target.
type Dispatcher struct {
	transport *Transport
	timeout   time.Duration
	rec       metric.Recorder
	openBodies atomic.Int64
}

// New creates a dispatcher with the given per-attempt timeout.
func New(transport *Transport, timeout time.Duration, rec metric.Recorder) *Dispatcher {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &Dispatcher{transport: transport, timeout: timeout, rec: rec}
}

// Forward sends one request to the target and returns the backend response.
// The outgoing HTTP request carries a deadline derived from the caller's
// context, so a slow backend can never hold the connection indefinitely.
func (d *Dispatcher) Forward(ctx context.Context, target model.Target, req *model.InferenceRequest) (model.Response, error) {
	httpReq, err := buildForwardRequest(context.Background(), target, req)
	if err != nil {
		return model.Response{}, err
	}
	resp, err := d.transport.Do(httpReq)
	if err != nil {
		d.rec.Inc(metric.NameDispatchError)
		return model.Response{}, err
	}
	d.openBodies.Add(1)
	defer func() {
		_ = d.transport.Close(resp)
		d.openBodies.Add(-1)
		d.rec.Set(metric.NameConnInFlight, d.openBodies.Load())
	}()
	preview := make([]byte, 4096)
	n, _ := resp.Body.Read(preview)
	d.rec.Inc(metric.NameDispatchOK)
	return model.Response{
		Status: resp.StatusCode,
		Body:   preview[:n],
		Target: target,
	}, nil
}

// Timeout returns the configured per-attempt timeout.
func (d *Dispatcher) Timeout() time.Duration {
	return d.timeout
}

// OpenBodies returns the number of response bodies that have not been
// released yet. It stays zero when every forwarded response is closed.
func (d *Dispatcher) OpenBodies() int64 {
	return d.openBodies.Load()
}
