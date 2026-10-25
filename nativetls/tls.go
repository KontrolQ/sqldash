package nativetls

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type managedCertificates struct {
	mutex       sync.Mutex
	options     Options
	host        string
	authority   *x509.Certificate
	certificate *tls.Certificate
}

func Configure(ctx context.Context, options Options) (*Material, error) {
	if options.AuthorityName == "" {
		options.AuthorityName = "sqldash native certificate authority"
	}
	if options.LeafName == "" {
		options.LeafName = "sqldash native"
	}
	if ctx == nil || ctx.Err() != nil {
		return nil, ErrInvalidTLS
	}
	host, hostError := certificateHost(options.Host)
	if hostError != nil {
		return nil, hostError
	}
	if (options.CertificatePath == "") != (options.KeyPath == "") {
		return nil, fmt.Errorf("%w: %s", ErrInvalidTLS, incompleteCertificatePair)
	}
	if options.CertificatePath != "" {
		certificateBytes, certificateError := readCertificateFile(options.CertificatePath)
		if certificateError != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidTLS, invalidCertificateMessage)
		}
		keyBytes, keyError := readCertificateFile(options.KeyPath)
		if keyError != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidTLS, invalidCertificateMessage)
		}
		defer clear(keyBytes)
		certificate, pairError := tls.X509KeyPair(certificateBytes, keyBytes)
		if pairError != nil || !validLeaf(&certificate, host, time.Now()) {
			return nil, fmt.Errorf("%w: %s", ErrInvalidTLS, invalidCertificateMessage)
		}
		configuration, configurationError := nativeTLSConfig(options.Config, options.ApplicationProtocols)
		if configurationError != nil {
			return nil, configurationError
		}
		configuration.Certificates = []tls.Certificate{certificate}
		configuration.GetCertificate = nil
		return &Material{Config: configuration}, nil
	}
	if options.ConfigurationDirectory == "" {
		return nil, ErrInvalidTLS
	}
	if directoryError := os.MkdirAll(options.ConfigurationDirectory, 0700); directoryError != nil {
		return nil, fmt.Errorf("%w: certificate storage", ErrInvalidTLS)
	}
	if options.AutomaticIssuer != nil {
		storageDirectory := filepath.Join(options.ConfigurationDirectory, "automatic")
		if directoryError := os.MkdirAll(storageDirectory, 0700); directoryError != nil {
			return nil, fmt.Errorf("%w: automatic certificate storage", ErrInvalidTLS)
		}
		issued, issuerError := options.AutomaticIssuer(ctx, host, storageDirectory)
		if issuerError != nil || issued == nil {
			return nil, fmt.Errorf("%w: automatic certificate issuance", ErrInvalidTLS)
		}
		configuration, configurationError := nativeTLSConfig(issued, options.ApplicationProtocols)
		if configurationError != nil || (len(configuration.Certificates) == 0 && configuration.GetCertificate == nil) {
			return nil, ErrInvalidTLS
		}
		for certificateIndex := range configuration.Certificates {
			if !validLeaf(&configuration.Certificates[certificateIndex], host, time.Now()) {
				return nil, fmt.Errorf("%w: %s", ErrInvalidTLS, invalidCertificateMessage)
			}
		}
		return &Material{Config: configuration}, nil
	}
	if options.LeafValidity == 0 {
		options.LeafValidity = DefaultLeafValidity
	}
	if options.RenewBefore == 0 {
		options.RenewBefore = DefaultRenewBefore
	}
	if options.LeafValidity <= 0 || options.RenewBefore < 0 || options.RenewBefore >= options.LeafValidity || options.LeafValidity > CertificateAuthorityValidity {
		return nil, ErrInvalidTLS
	}
	release, lockError := lockCertificateStorage(options.ConfigurationDirectory)
	if lockError != nil {
		return nil, lockError
	}
	defer release()
	authority, authorityKey, publicAuthority, authorityError := loadOrCreateAuthority(options.ConfigurationDirectory, options.AuthorityName)
	if authorityError != nil {
		return nil, authorityError
	}
	certificate, certificateError := loadOrCreateLeaf(options, host, authority, authorityKey)
	if certificateError != nil {
		return nil, certificateError
	}
	configuration, configurationError := nativeTLSConfig(options.Config, options.ApplicationProtocols)
	if configurationError != nil {
		return nil, configurationError
	}
	managed := &managedCertificates{options: options, host: host, authority: authority, certificate: certificate}
	configuration.Certificates = nil
	configuration.GetCertificate = managed.getCertificate
	fingerprint := sha256.Sum256(authority.Raw)
	encodedFingerprint := strings.ToUpper(hex.EncodeToString(fingerprint[:]))
	parts := make([]string, 0, len(fingerprint))
	for position := 0; position < len(encodedFingerprint); position += 2 {
		parts = append(parts, encodedFingerprint[position:position+2])
	}
	return &Material{Config: configuration, CACertificate: publicAuthority, CAFingerprint: strings.Join(parts, ":")}, nil
}

func nativeTLSConfig(existing *tls.Config, protocols []string) (*tls.Config, error) {
	configuration := &tls.Config{}
	if existing != nil {
		configuration = existing.Clone()
	}
	if configuration.GetConfigForClient != nil || (configuration.MaxVersion != 0 && configuration.MaxVersion < tls.VersionTLS12) {
		return nil, ErrInvalidTLS
	}
	if configuration.MinVersion < tls.VersionTLS12 {
		configuration.MinVersion = tls.VersionTLS12
	}
	if configuration.MaxVersion != 0 && configuration.MaxVersion < configuration.MinVersion {
		return nil, ErrInvalidTLS
	}
	configuration.NextProtos = append([]string(nil), protocols...)
	return configuration, nil
}

func (self *managedCertificates) getCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if hello.Context().Err() != nil {
		return nil, hello.Context().Err()
	}
	now := time.Now()
	if now.Before(self.authority.NotBefore) || !now.Before(self.authority.NotAfter) || now.Before(self.certificate.Leaf.NotBefore) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidTLS, invalidAuthorityMessage)
	}
	if self.certificate.Leaf.NotAfter.Sub(now) > self.options.RenewBefore {
		return self.certificate, nil
	}
	release, lockError := lockCertificateStorage(self.options.ConfigurationDirectory)
	if lockError != nil {
		return nil, lockError
	}
	defer release()
	authority, authorityKey, _, authorityError := readAuthority(self.options.ConfigurationDirectory)
	if authorityError != nil || !self.authority.Equal(authority) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidTLS, invalidAuthorityMessage)
	}
	certificate, certificateError := loadOrCreateLeaf(self.options, self.host, authority, authorityKey)
	if certificateError != nil {
		return nil, certificateError
	}
	self.certificate = certificate
	return certificate, nil
}

func validLeaf(certificate *tls.Certificate, host string, now time.Time) bool {
	if len(certificate.Certificate) == 0 {
		return false
	}
	leaf, parseError := x509.ParseCertificate(certificate.Certificate[0])
	if parseError != nil || leaf.IsCA || now.Before(leaf.NotBefore) || !now.Before(leaf.NotAfter) || leaf.VerifyHostname(host) != nil {
		return false
	}
	certificate.Leaf = leaf
	return true
}

func certificateHost(value string) (string, error) {
	value = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), ".")
	if value == "" {
		return "", ErrInvalidTLS
	}
	if address := net.ParseIP(strings.Trim(value, "[]")); address != nil {
		return address.String(), nil
	}
	if len(value) > 253 {
		return "", ErrInvalidTLS
	}
	for _, label := range strings.Split(value, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", ErrInvalidTLS
		}
		for _, character := range label {
			if !(character >= 'a' && character <= 'z') && !(character >= '0' && character <= '9') && character != '-' {
				return "", ErrInvalidTLS
			}
		}
	}
	return value, nil
}
