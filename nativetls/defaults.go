package nativetls

import "time"

const (
	DefaultLeafValidity          = 365 * 24 * time.Hour
	DefaultRenewBefore           = 30 * 24 * time.Hour
	CertificateAuthorityValidity = 10 * 365 * 24 * time.Hour
)
