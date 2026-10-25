package redisconnections

import "time"

const (
	pendingNamespace   = "redis-credential-pending"
	pendingLifetime    = 15 * time.Minute
	maximumCredentials = 1000
	stateSuspended     = "suspended"
	stateRevoked       = "revoked"
	stateDelivering    = "delivering"
	stateStale         = "stale"
)
