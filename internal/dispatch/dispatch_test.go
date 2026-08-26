package dispatch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"modelrouter/internal/metric"
	"modelrouter/internal/model"
)

func TestForwardToFastBackend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()
	transport := NewTransport(server.Client(), metric.NullRecorder{})
	dispatcher := New(transport, time.Second, metric.NullRecorder{})
	target := model.Target{Model: "resnet", Version: "v1", InstanceID: "a", Addr: server.Listener.Addr().String()}
	resp, err := dispatcher.Forward(context.Background(), target, &model.InferenceRequest{
		Model:     "resnet",
		RequestID: "req-1",
		Payload:   []byte("hello"),
	})
	if err != nil {
		t.Fatalf("forward: %v", err)
	}
	if string(resp.Body) != "ok" || resp.Status != http.StatusOK {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestForwardSendsModelHeaders(t *testing.T) {
	var gotModel, gotVersion, gotInstance string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotModel = r.Header.Get("X-Model")
		gotVersion = r.Header.Get("X-Version")
		gotInstance = r.Header.Get("X-Instance")
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()
	transport := NewTransport(server.Client(), metric.NullRecorder{})
	dispatcher := New(transport, time.Second, metric.NullRecorder{})
	target := model.Target{Model: "resnet", Version: "v1", InstanceID: "a", Addr: server.Listener.Addr().String()}
	_, err := dispatcher.Forward(context.Background(), target, &model.InferenceRequest{
		Model:     "resnet",
		RequestID: "req-2",
		Payload:   []byte("hello"),
	})
	if err != nil {
		t.Fatalf("forward: %v", err)
	}
	if gotModel != "resnet" || gotVersion != "v1" || gotInstance != "a" {
		t.Fatalf("headers: model=%q version=%q instance=%q", gotModel, gotVersion, gotInstance)
	}
}

func TestTransportCloseIsIdempotent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()
	transport := NewTransport(server.Client(), metric.NullRecorder{})
	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := transport.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	if err := transport.Close(resp); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := transport.Close(nil); err != nil {
		t.Fatalf("close nil: %v", err)
	}
}

func TestDeadlineAndRemaining(t *testing.T) {
	ctx, cancel := Deadline(context.Background(), 100*time.Millisecond)
	defer cancel()
	remaining := Remaining(ctx)
	if remaining <= 0 || remaining > 100*time.Millisecond {
		t.Fatalf("unexpected remaining: %v", remaining)
	}
	if Remaining(context.Background()) != 0 {
		t.Fatal("background context must have no remaining deadline")
	}
}

func TestTimeoutDefault(t *testing.T) {
	transport := NewTransport(http.DefaultClient, metric.NullRecorder{})
	dispatcher := New(transport, 0, metric.NullRecorder{})
	if dispatcher.Timeout() != 5*time.Second {
		t.Fatalf("timeout = %v, want 5s", dispatcher.Timeout())
	}
}
