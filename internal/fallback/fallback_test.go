package fallback

import (
	"context"
	"testing"

	"modelrouter/internal/metric"
	"modelrouter/internal/model"
)

func TestPolicyValidation(t *testing.T) {
	if err := DefaultPolicy().Validate(); err != nil {
		t.Fatalf("default policy must validate: %v", err)
	}
	if err := (Policy{MaxAttempts: 0}).Validate(); err == nil {
		t.Fatal("zero-attempt policy must be rejected")
	}
}

func TestExecuteReturnsFirstSuccess(t *testing.T) {
	executor, err := New(Policy{MaxAttempts: 3}, metric.NullRecorder{})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	targets := []model.Target{
		{Model: "resnet", Version: "v1", InstanceID: "a"},
		{Model: "resnet", Version: "v1", InstanceID: "b"},
	}
	resp, target, err := executor.Execute(context.Background(), targets, func(_ context.Context, t model.Target) (model.Response, error) {
		return model.Response{Status: 200, Body: []byte("pong"), Target: t}, nil
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if target.InstanceID != "a" || string(resp.Body) != "pong" {
		t.Fatalf("unexpected result: %+v %+v", resp, target)
	}
}

func TestExecuteSkipsNonSuccessStatus(t *testing.T) {
	executor, err := New(Policy{MaxAttempts: 2}, metric.NullRecorder{})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	targets := []model.Target{
		{Model: "resnet", Version: "v1", InstanceID: "a"},
		{Model: "resnet", Version: "v1", InstanceID: "b"},
	}
	resp, target, err := executor.Execute(context.Background(), targets, func(_ context.Context, t model.Target) (model.Response, error) {
		if t.InstanceID == "a" {
			return model.Response{Status: 503, Body: []byte("down"), Target: t}, nil
		}
		return model.Response{Status: 200, Body: []byte("ok"), Target: t}, nil
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if target.InstanceID != "b" {
		t.Fatalf("fallback must skip non-success target, got %s", target.InstanceID)
	}
	if !resp.Success() {
		t.Fatalf("final response must be successful: %+v", resp)
	}
}

func TestExecutePolicyReturnsPolicy(t *testing.T) {
	policy := Policy{MaxAttempts: 2}
	executor, err := New(policy, metric.NullRecorder{})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if executor.Policy().MaxAttempts != 2 {
		t.Fatalf("policy = %+v", executor.Policy())
	}
}
