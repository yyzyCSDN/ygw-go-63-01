package gateway

import (
	"context"
	"errors"
	"time"

	"modelrouter/internal/model"
)

// ServeTimeout is the default overall deadline applied to a request when
// the caller does not provide one.
const ServeTimeout = 10 * time.Second

// ServeWithTimeout is Serve with an explicit overall deadline.
func (g *Gateway) ServeWithTimeout(
	parent context.Context,
	modelName, requestID string,
	payload []byte,
	deadline time.Duration,
) (model.Response, error) {
	if deadline <= 0 {
		deadline = ServeTimeout
	}
	ctx, cancel := context.WithTimeout(parent, deadline)
	defer cancel()
	return g.Serve(ctx, modelName, requestID, payload)
}

// MapError converts a domain error into an HTTP-style status code.
func MapError(err error) int {
	switch {
	case err == nil:
		return 200
	case errors.Is(err, model.ErrModelNotFound), errors.Is(err, model.ErrVersionNotFound), errors.Is(err, model.ErrAliasNotFound):
		return 404
	case errors.Is(err, model.ErrQuotaExceeded):
		return 429
	case errors.Is(err, model.ErrSyncFailed):
		return 503
	case errors.Is(err, model.ErrTargetUnavailable):
		return 502
	default:
		return 500
	}
}
