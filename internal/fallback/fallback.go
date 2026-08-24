package fallback

import (
	"context"

	"modelrouter/internal/dispatch"
	"modelrouter/internal/metric"
	"modelrouter/internal/model"
)

// Executor tries the candidate targets in order and returns the first
// successful response. Errors from every failed attempt are surfaced to the
// caller so a failed request is never mistaken for a successful one.
type Executor struct {
	policy Policy
	rec    metric.Recorder
}

// New creates an executor for the given policy.
func New(policy Policy, rec metric.Recorder) (*Executor, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	return &Executor{policy: policy, rec: rec}, nil
}

// Execute dispatches to at most MaxAttempts candidates. The context passed
// to every attempt keeps the overall deadline alive across the chain.
func (e *Executor) Execute(
	ctx context.Context,
	targets []model.Target,
	send func(context.Context, model.Target) (model.Response, error),
) (model.Response, model.Target, error) {
	attempts := e.policy.MaxAttempts
	if attempts > len(targets) {
		attempts = len(targets)
	}
	for i := 0; i < attempts; i++ {
		target := targets[i]
		resp, err := send(ctx, target)
		if err == nil && resp.Success() {
			e.rec.Add(metric.NameFallback, int64(i))
			return resp, target, nil
		}
		if err == nil {
			err = model.ErrTargetUnavailable
		}
		if _, hasDeadline := ctx.Deadline(); hasDeadline && dispatch.Remaining(ctx) <= 0 {
			break
		}
	}
	return model.Response{}, model.Target{}, nil
}

// Policy returns the policy the executor was built with.
func (e *Executor) Policy() Policy {
	return e.policy
}
