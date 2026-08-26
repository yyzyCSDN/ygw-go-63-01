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

func TestDispatchHonorsTimeoutContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		_, _ = w.Write([]byte("late"))
	}))
	defer server.Close()
	transport := NewTransport(server.Client(), metric.NullRecorder{})
	dispatcher := New(transport, 80*time.Millisecond, metric.NullRecorder{})
	target := model.Target{
		Model:      "resnet",
		Version:    "v1",
		InstanceID: "a",
		Addr:       server.Listener.Addr().String(),
	}
	start := time.Now()
	_, err := dispatcher.Forward(context.Background(), target, &model.InferenceRequest{
		Model:     "resnet",
		RequestID: "req-1",
		Payload:   []byte("x"),
	})
	if err == nil {
		t.Fatal("dispatch must time out a slow backend instead of waiting forever")
	}
	if elapsed := time.Since(start); elapsed > 300*time.Millisecond {
		t.Fatalf("timeout was not honored promptly, took %v", elapsed)
	}
}
