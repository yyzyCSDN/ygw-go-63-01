package model

import "errors"

// Sentinel errors shared by every component of the gateway.
var (
	ErrModelNotFound     = errors.New("model not found")
	ErrVersionNotFound   = errors.New("version not found")
	ErrAliasNotFound     = errors.New("alias not found")
	ErrQuotaExceeded     = errors.New("quota exceeded")
	ErrTargetUnavailable = errors.New("target unavailable")
	ErrSyncFailed        = errors.New("registry sync failed")
	ErrConflict          = errors.New("conflicting state")
)
