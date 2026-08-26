package dispatch

import (
	"net/http"

	"modelrouter/internal/metric"
)

// Transport wraps the shared HTTP client used for forwarding.
type Transport struct {
	client *http.Client
	rec    metric.Recorder
}

// NewTransport creates a transport over the given client.
func NewTransport(client *http.Client, rec metric.Recorder) *Transport {
	return &Transport{client: client, rec: rec}
}

// Do performs one HTTP request.
func (t *Transport) Do(req *http.Request) (*http.Response, error) {
	return t.client.Do(req)
}

// Close drains a bounded amount of the response body and closes it so the
// underlying connection returns to the pool. Draining a bounded prefix keeps
// reuse predictable without copying unbounded payloads.
func (t *Transport) Close(resp *http.Response) error {
	if resp == nil || resp.Body == nil {
		return nil
	}
	return resp.Body.Close()
}
