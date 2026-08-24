package gateway

import (
	"context"
	"fmt"
	"time"

	"modelrouter/internal/dispatch"
	"modelrouter/internal/fallback"
	"modelrouter/internal/metric"
	"modelrouter/internal/model"
	"modelrouter/internal/quota"
	"modelrouter/internal/registry"
	"modelrouter/internal/route"
)

// Gateway orchestrates the full request path: route resolution, quota
// admission, dispatch and fallback.
type Gateway struct {
	reg        *registry.Registry
	table      *route.Table
	limiter    *quota.Limiter
	dispatcher *dispatch.Dispatcher
	fallback   *fallback.Executor
	rec        metric.Recorder
}

// New wires the gateway components together.
func New(
	reg *registry.Registry,
	table *route.Table,
	limiter *quota.Limiter,
	dispatcher *dispatch.Dispatcher,
	executor *fallback.Executor,
	rec metric.Recorder,
) *Gateway {
	return &Gateway{
		reg:        reg,
		table:      table,
		limiter:    limiter,
		dispatcher: dispatcher,
		fallback:   executor,
		rec:        rec,
	}
}

// Serve handles one inference request end to end.
func (g *Gateway) Serve(ctx context.Context, modelName, requestID string, payload []byte) (model.Response, error) {
	result, err := g.table.Resolve(modelName, requestID)
	if err != nil {
		return model.Response{}, err
	}
	slot, err := g.limiter.Acquire()
	if err != nil {
		g.rec.Inc(metric.NameQuotaReject)
		return model.Response{}, err
	}
	defer func() {
		_ = g.limiter.Release(slot)
	}()
	req := &model.InferenceRequest{
		Model:     result.Model,
		RequestID: requestID,
		Payload:   payload,
	}
	candidates := result.Candidates()
	resp, _, err := g.fallback.Execute(ctx, candidates, func(inner context.Context, t model.Target) (model.Response, error) {
		return g.dispatcher.Forward(inner, t, req)
	})
	if err != nil {
		return model.Response{}, err
	}
	return resp, nil
}

// Timeout returns the configured per-attempt dispatch timeout.
func (g *Gateway) Timeout() time.Duration {
	return g.dispatcher.Timeout()
}

// FallbackPolicy exposes the fallback policy for diagnostics.
func (g *Gateway) FallbackPolicy() fallback.Policy {
	return g.fallback.Policy()
}

// Registry exposes the underlying registry for admin operations.
func (g *Gateway) Registry() *registry.Registry {
	return g.reg
}

// Table exposes the route table for diagnostics.
func (g *Gateway) Table() *route.Table {
	return g.table
}

// Limiter exposes the quota limiter for diagnostics.
func (g *Gateway) Limiter() *quota.Limiter {
	return g.limiter
}

// String returns a short description of the gateway wiring.
func (g *Gateway) String() string {
	return fmt.Sprintf("gateway(models=%d, quota=%d)", len(g.reg.Models()), g.limiter.Capacity())
}
