package nativetls

import "errors"

const (
	invalidCertificateMessage = "invalid native certificate or private key"
	incompleteCertificatePair = "native certificate and private key must both be supplied"
	invalidAuthorityMessage   = "native certificate authority is missing, damaged, or expired"
	certificateStorageBusy    = "native certificate storage is already being changed"
)

var ErrInvalidTLS = errors.New("invalid native TLS configuration")
