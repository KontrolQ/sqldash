package redisgateway

import "errors"

var (
	InvalidOptions       = errors.New("invalid Redis gateway configuration")
	AccessDenied         = errors.New("Redis connection is not authorized")
	AuthenticationFailed = errors.New("Redis authentication failed")
	RebindingDenied      = errors.New("Redis connection binding cannot be changed")
	BackendUnavailable   = errors.New("Redis backend is unavailable")
	EnforcementFailed    = errors.New("Redis backend termination could not be confirmed")
)

const (
	authenticationRequired = "-NOAUTH Named Redis authentication is required.\r\n"
	authenticationDenied   = "-WRONGPASS Invalid username-password pair or user is disabled.\r\n"
	tlsRequired            = "-NOAUTH TLS is required for Redis connections.\r\n"
	backendUnavailable     = "-ERR Redis database is unavailable.\r\n"
	connectionLimit        = "-ERR Redis connection limit reached.\r\n"
)
