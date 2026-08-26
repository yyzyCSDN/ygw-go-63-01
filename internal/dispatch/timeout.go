package dispatch

import (
	"context"
	"time"
)

// Deadline derives the per-attempt deadline from a request context and a
// configured timeout. The result context is what the outgoing HTTP request
// must carry so that cancellation propagates to the backend connection.
func Deadline(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// Remaining returns the time left on a context deadline, or zero when no
// deadline is set.
func Remaining(ctx context.Context) time.Duration {
	deadline, ok := ctx.Deadline()
	if !ok {
		return 0
	}
	left := time.Until(deadline)
	if left < 0 {
		return 0
	}
	return left
}
