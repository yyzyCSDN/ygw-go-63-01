package fallback

import (
	"context"
	"fmt"
	"log"

	"modelrouter/internal/dispatch"
	"modelrouter/internal/metric"
	"modelrouter/internal/model"
)

// Executor tries the candidate targets in order and returns the first
// successful response. Errors from every failed attempt are recorded and
// reported so a failed request is never mistaken for a successful one. When
// no candidate succeeds the executor returns an error wrapping
// model.ErrTargetUnavailable instead of a nil error.
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
// to every attempt keeps the overall deadline alive across the chain. Each
// failed attempt is logged and counted before the executor moves on to the
// next healthy candidate; only when every attempt has failed does Execute
// return an error.
func (e *Executor) Execute(
	ctx context.Context,
	targets []model.Target,
	send func(context.Context, model.Target) (model.Response, error),
) (model.Response, model.Target, error) {
	attempts := e.policy.MaxAttempts
	if attempts > len(targets) {
		attempts = len(targets)
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		target := targets[i]
		resp, err := send(ctx, target)
		if err == nil && resp.Success() {
			e.rec.Add(metric.NameFallback, int64(i))
			return resp, target, nil
		}
		if err == nil {
			err = fmt.Errorf("%w: %s returned status %d", model.ErrTargetUnavailable, target.InstanceID, resp.Status)
		}
		lastErr = err
		log.Printf("fallback: attempt %d on %s/%s instance %s failed: %v",
			i+1, target.Model, target.Version, target.InstanceID, err)
		e.rec.Inc(metric.NameFallbackError)
		if _, hasDeadline := ctx.Deadline(); hasDeadline && dispatch.Remaining(ctx) <= 0 {
			break
		}
	}
	if lastErr == nil {
		lastErr = model.ErrTargetUnavailable
	}
	return model.Response{}, model.Target{}, fmt.Errorf("%w: no candidate succeeded after %d attempt(s)", model.ErrTargetUnavailable, attempts)
}

// Policy returns the policy the executor was built with.
func (e *Executor) Policy() Policy {
	return e.policy
}
