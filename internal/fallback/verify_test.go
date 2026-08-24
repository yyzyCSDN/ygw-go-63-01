package fallback

import (
	"context"
	"testing"
	"time"

	"modelrouter/internal/metric"
	"modelrouter/internal/model"
)

func TestFallbackErrorNotSwallowed(t *testing.T) {
	executor, err := New(Policy{MaxAttempts: 2}, metric.NullRecorder{})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	targets := []model.Target{
		{Model: "resnet", Version: "v1", InstanceID: "a"},
		{Model: "resnet", Version: "v1", InstanceID: "b"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, _, err = executor.Execute(ctx, targets, func(_ context.Context, target model.Target) (model.Response, error) {
		return model.Response{}, model.ErrTargetUnavailable
	})
	if err == nil {
		t.Fatal("fallback failure must be surfaced to the caller")
	}
}
