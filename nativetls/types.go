package nativetls

import (
	"context"
	"crypto/tls"
	"time"
)

type Options struct {
	ConfigurationDirectory string
	Host                   string
	CertificatePath        string
	KeyPath                string
	Config                 *tls.Config
	AutomaticIssuer        func(context.Context, string, string) (*tls.Config, error)
	LeafValidity           time.Duration
	RenewBefore            time.Duration
	AuthorityName          string
	LeafName               string
	ApplicationProtocols   []string
}

type Material struct {
	Config        *tls.Config
	CACertificate []byte
	CAFingerprint string
}
