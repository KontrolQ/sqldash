package pggateway

import "time"

const (
	LogPrefix = "PostgreSQL gateway"

	DefaultMaximumConnections         = 256
	DefaultMaximumSessions            = 128
	DefaultMaximumHandshakes          = 32
	DefaultMaximumStartupBytes        = 10 * 1024
	DefaultMaximumHandshakeFrameBytes = 64 * 1024
	DefaultMaximumHandshakeBytes      = 1024 * 1024
	DefaultMaximumCancelRequests      = 32
	DefaultMaximumSessionCancels      = 8
	DefaultHandshakeTimeout           = 30 * time.Second
	DefaultTLSHandshakeTimeout        = 10 * time.Second
	DefaultBackendConnectTimeout      = 5 * time.Second
	DefaultCancelTimeout              = 5 * time.Second
	DefaultValidationInterval         = 2 * time.Second
	DefaultValidationTimeout          = 2 * time.Second
	DefaultTerminationTimeout         = 5 * time.Second
	MaximumPolicyDelay                = 5 * time.Second
	DefaultLeafValidity               = 365 * 24 * time.Hour
	DefaultRenewBefore                = 30 * 24 * time.Hour
	CertificateAuthorityValidity      = 10 * 365 * 24 * time.Hour

	protocol30                    uint32 = 196608
	protocol32                    uint32 = 196610
	sslRequestCode                uint32 = 80877103
	gssRequestCode                uint32 = 80877104
	cancelRequestCode             uint32 = 80877102
	minimumCancelKeyBytes                = 4
	maximumCancelKeyBytes                = 256
	maximumCancelPacketBytes             = 12 + maximumCancelKeyBytes
	postgresqlApplicationProtocol        = "postgresql"
)
