package pggateway

import "errors"

const (
	invalidConfigurationMessage = "invalid PostgreSQL gateway configuration"
	invalidStartupMessage       = "invalid PostgreSQL startup message"
	unsupportedProtocolMessage  = "unsupported PostgreSQL protocol version"
	tlsRequiredMessage          = "TLS is required for PostgreSQL connections"
	accessDeniedMessage         = "PostgreSQL connection is not authorized"
	backendUnavailableMessage   = "PostgreSQL is unavailable"
	handshakeFailedMessage      = "PostgreSQL authentication failed"
	connectionLimitMessage      = "PostgreSQL connection limit reached"
	channelBindingMessage       = "SCRAM channel binding is not supported by this gateway; use channel_binding=disable or prefer"
	terminationFailedMessage    = "PostgreSQL backend termination could not be confirmed"
	invalidTLSConfiguration     = "invalid PostgreSQL TLS configuration"
	invalidCertificateMessage   = "invalid PostgreSQL certificate or private key"
	incompleteCertificatePair   = "PostgreSQL certificate and private key must both be supplied"
	invalidAuthorityMessage     = "PostgreSQL certificate authority is missing, damaged, or expired"
	certificateStorageBusy      = "PostgreSQL certificate storage is already being changed"
	callbackFailedMessage       = "PostgreSQL gateway callback did not complete normally"
	certificateAuthorityName    = "sqldash PostgreSQL certificate authority"
	certificateLeafName         = "sqldash PostgreSQL"
)

var (
	ErrInvalidOptions      = errors.New(invalidConfigurationMessage)
	ErrInvalidStartup      = errors.New(invalidStartupMessage)
	ErrUnsupportedProtocol = errors.New(unsupportedProtocolMessage)
	ErrTLSRequired         = errors.New(tlsRequiredMessage)
	ErrAccessDenied        = errors.New(accessDeniedMessage)
	ErrBackendUnavailable  = errors.New(backendUnavailableMessage)
	ErrAuthentication      = errors.New(handshakeFailedMessage)
	ErrConnectionLimit     = errors.New(connectionLimitMessage)
	ErrChannelBinding      = errors.New(channelBindingMessage)
	ErrUnsafe              = errors.New(terminationFailedMessage)
	ErrInvalidTLS          = errors.New(invalidTLSConfiguration)
	ErrCallback            = errors.New(callbackFailedMessage)
)
