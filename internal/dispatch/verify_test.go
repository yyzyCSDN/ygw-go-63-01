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

func TestDispatchConnClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()
	transport := NewTransport(server.Client(), metric.NullRecorder{})
	dispatcher := New(transport, time.Second, metric.NullRecorder{})
	target := model.Target{
		Model:      "resnet",
		Version:    "v1",
		InstanceID: "a",
		Addr:       server.Listener.Addr().String(),
	}
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		resp, err := dispatcher.Forward(ctx, target, &model.InferenceRequest{
			Model:     "resnet",
			RequestID: "req-" + string(rune('0'+i)),
			Payload:   []byte("x"),
		})
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		if string(resp.Body) != "ok" {
			t.Fatalf("request %d body = %q", i, resp.Body)
		}
		if open := dispatcher.OpenBodies(); open != 0 {
			t.Fatalf("request %d left %d response bodies open; forwarded connections must be released", i, open)
		}
	}
}
