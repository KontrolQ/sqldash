package redisgateway

import "time"

const (
	DefaultMaximumConnections         = 256
	DefaultMaximumSessionsPerDatabase = 100
	DefaultMaximumHandshakes          = 32
	DefaultHandshakeTimeout           = 10 * time.Second
	DefaultValidationInterval         = 2 * time.Second
	DefaultValidationTimeout          = time.Second
	DefaultTerminationTimeout         = 2 * time.Second
	MaximumPolicyDelay                = 5 * time.Second
	maximumAuthenticationBytes        = 16 * 1024
	maximumAuthenticationCommands     = 4
)
