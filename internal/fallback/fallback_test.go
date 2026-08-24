package fallback

import (
	"context"
	"errors"
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

func TestExecuteAllFailReturnsError(t *testing.T) {
	executor, err := New(Policy{MaxAttempts: 3}, metric.NullRecorder{})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	targets := []model.Target{
		{Model: "resnet", Version: "v1", InstanceID: "a"},
		{Model: "resnet", Version: "v1", InstanceID: "b"},
		{Model: "resnet", Version: "v1", InstanceID: "c"},
	}
	resp, target, err := executor.Execute(context.Background(), targets, func(_ context.Context, t model.Target) (model.Response, error) {
		return model.Response{Status: 503, Body: []byte("down"), Target: t}, nil
	})
	if err == nil {
		t.Fatal("exhausted fallback must surface an error, got nil")
	}
	if !errors.Is(err, model.ErrTargetUnavailable) {
		t.Fatalf("error must wrap ErrTargetUnavailable, got %v", err)
	}
	if target.InstanceID != "" {
		t.Fatalf("target must be zero value on failure, got %+v", target)
	}
	if resp.Status != 0 || resp.Body != nil {
		t.Fatalf("response must be zero value on failure, got %+v", resp)
	}
}

func TestExecuteReportsEachFailure(t *testing.T) {
	metrics := metric.New()
	rec := metric.NewRecorder(metrics)
	executor, err := New(Policy{MaxAttempts: 3}, rec)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	targets := []model.Target{
		{Model: "resnet", Version: "v1", InstanceID: "a"},
		{Model: "resnet", Version: "v1", InstanceID: "b"},
		{Model: "resnet", Version: "v1", InstanceID: "c"},
	}
	_, _, err = executor.Execute(context.Background(), targets, func(_ context.Context, t model.Target) (model.Response, error) {
		return model.Response{Status: 500, Body: []byte("boom"), Target: t}, nil
	})
	if err == nil {
		t.Fatal("exhausted fallback must surface an error, got nil")
	}
	if got := metrics.Counter(metric.NameFallbackError).Value(); got != 3 {
		t.Fatalf("fallback.error counter = %d, want 3 (one per failed attempt)", got)
	}
}

func TestExecuteSendErrorIsReportedAndSkipped(t *testing.T) {
	metrics := metric.New()
	rec := metric.NewRecorder(metrics)
	executor, err := New(Policy{MaxAttempts: 2}, rec)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	targets := []model.Target{
		{Model: "resnet", Version: "v1", InstanceID: "a"},
		{Model: "resnet", Version: "v1", InstanceID: "b"},
	}
	sentinel := errors.New("dial failed")
	_, _, err = executor.Execute(context.Background(), targets, func(_ context.Context, t model.Target) (model.Response, error) {
		return model.Response{}, sentinel
	})
	if err == nil {
		t.Fatal("exhausted fallback must surface an error, got nil")
	}
	if got := metrics.Counter(metric.NameFallbackError).Value(); got != 2 {
		t.Fatalf("fallback.error counter = %d, want 2 (send errors count as failures)", got)
	}
}

func TestExecuteReportsOnlyFailuresBeforeSuccess(t *testing.T) {
	metrics := metric.New()
	rec := metric.NewRecorder(metrics)
	executor, err := New(Policy{MaxAttempts: 3}, rec)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	targets := []model.Target{
		{Model: "resnet", Version: "v1", InstanceID: "a"},
		{Model: "resnet", Version: "v1", InstanceID: "b"},
		{Model: "resnet", Version: "v1", InstanceID: "c"},
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
	if target.InstanceID != "b" || !resp.Success() {
		t.Fatalf("unexpected result: target=%+v resp=%+v", target, resp)
	}
	if got := metrics.Counter(metric.NameFallbackError).Value(); got != 1 {
		t.Fatalf("fallback.error counter = %d, want 1 (only the failed first attempt)", got)
	}
	if got := metrics.Counter(metric.NameFallback).Value(); got != 1 {
		t.Fatalf("fallback.execute counter = %d, want 1", got)
	}
}
