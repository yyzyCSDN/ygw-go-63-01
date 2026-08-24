package gateway

import (
	"errors"
	"testing"
	"time"

	"modelrouter/internal/model"
)

func TestMapError(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{nil, 200},
		{model.ErrModelNotFound, 404},
		{model.ErrVersionNotFound, 404},
		{model.ErrAliasNotFound, 404},
		{model.ErrQuotaExceeded, 429},
		{model.ErrSyncFailed, 503},
		{model.ErrTargetUnavailable, 502},
		{errors.New("other"), 500},
	}
	for _, tc := range cases {
		if got := MapError(tc.err); got != tc.want {
			t.Fatalf("MapError(%v) = %d, want %d", tc.err, got, tc.want)
		}
	}
}

func TestServeTimeoutConstant(t *testing.T) {
	if ServeTimeout != 10*time.Second {
		t.Fatalf("ServeTimeout = %v", ServeTimeout)
	}
}
