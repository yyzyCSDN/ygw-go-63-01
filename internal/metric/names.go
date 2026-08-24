package metric

// Metric names shared across components.
const (
	NamePublish        = "registry.publish"
	NameRoute          = "route.resolve"
	NameFallback       = "fallback.execute"
	NameFallbackError  = "fallback.error"
	NameQuotaReject    = "quota.reject"
	NameDispatchError  = "dispatch.error"
	NameDispatchOK     = "dispatch.ok"
	NameConnInFlight   = "dispatch.conn_in_flight"
	NameSync           = "registry.sync"
	NameAlias          = "registry.alias"
	NameHealthObserved = "health.observed"
)
