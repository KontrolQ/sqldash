package pggateway

import (
	"errors"
	"sqldash/nativetls"
)

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
	ErrInvalidTLS          = nativetls.ErrInvalidTLS
	ErrCallback            = errors.New(callbackFailedMessage)
)
