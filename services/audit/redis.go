package audit

const (
	RedisCredentialViewed   = "redis.connection.viewed"
	RedisCredentialCreated  = "redis.connection.created"
	RedisCredentialRotated  = "redis.connection.rotated"
	RedisCredentialRevoked  = "redis.connection.revoked"
	RedisCredentialRevealed = "redis.connection.revealed"
	RedisClientTerminated   = "redis.activity.terminated"
)

func init() {
	labels[RedisCredentialViewed] = "Redis credentials viewed"
	labels[RedisCredentialCreated] = "Redis credential created"
	labels[RedisCredentialRotated] = "Redis password rotated"
	labels[RedisCredentialRevoked] = "Redis credential revoked"
	labels[RedisCredentialRevealed] = "Redis one-time credential delivered"
	labels[RedisClientTerminated] = "Redis client disconnected"
}
