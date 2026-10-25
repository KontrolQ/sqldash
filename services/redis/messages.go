package redis

const (
	AuthorityRefused = "Your account no longer has permission for this Redis action."
	DatabaseMissing  = "This Redis database is not available to your account."
	DatabaseNotReady = "This Redis database is not ready. Retry after its lifecycle operation finishes."
	RedisUnavailable = "Redis is unavailable. Your changes have not been discarded; reconnect before retrying."
	InvalidRequest   = "The Redis request is invalid or exceeds the permitted size."
	ReadOnly         = "Read-only access: this account cannot change Redis data."
)
