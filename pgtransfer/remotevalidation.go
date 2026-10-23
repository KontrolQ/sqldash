package pgtransfer

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"net"
	"strings"
	"unicode"
	"unicode/utf8"

	"sqldash/pgcluster/control"
)

func ValidateRemoteSource(source RemoteSource) error {
	if !validRemoteHost(source.Host) || source.Port < 1 || source.Port > 65535 || !validRemoteIdentifier(source.Database) || !validRemoteIdentifier(source.Username) || strings.Contains(source.Database, "=") || strings.HasPrefix(strings.ToLower(source.Database), "postgresql://") || strings.HasPrefix(strings.ToLower(source.Database), "postgres://") || len(source.Password) > MaximumRemotePasswordBytes || !utf8.ValidString(source.Password) || strings.ContainsAny(source.Password, "\r\n\x00") {
		return InvalidRemoteSource
	}
	mode := remoteSSLMode(source)
	if len(source.RootCertificate) > MaximumCertificateBytes {
		return InvalidRemoteTLS
	}
	switch mode {
	case SSLVerifyFull, SSLVerifyCA:
	case SSLRequire, SSLDisable:
		if len(source.RootCertificate) != 0 {
			return InvalidRemoteTLS
		}
	default:
		return InvalidRemoteTLS
	}
	if len(source.RootCertificate) != 0 {
		if _, certificateError := certificatePool(source.RootCertificate); certificateError != nil {
			return certificateError
		}
	}
	if address := net.ParseIP(source.Host); address != nil && remotePrivateAddress(address, source.Port) {
		return RemoteSourceDenied
	}
	return nil
}

func validRemoteIdentifier(value string) bool {
	return validRemoteText(value, 63)
}

func validRemoteText(value string, maximumBytes int) bool {
	return value != "" && len(value) <= maximumBytes && utf8.ValidString(value) && strings.IndexFunc(value, unicode.IsControl) == -1
}

func validRemoteHost(host string) bool {
	if host == "" || len(host) > 253 || strings.IndexFunc(host, unicode.IsControl) != -1 || strings.ContainsAny(host, " /\\,@=[]%\t\r\n") {
		return false
	}
	if address := net.ParseIP(host); address != nil {
		return !address.IsUnspecified() && !address.IsMulticast()
	}
	for _, label := range strings.Split(strings.TrimSuffix(host, "."), ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, character := range label {
			if character != '-' && (character < '0' || character > '9') && (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') {
				return false
			}
		}
	}
	return true
}

func remoteSSLMode(source RemoteSource) string {
	if source.SSLMode == "" {
		return SSLVerifyFull
	}
	return source.SSLMode
}

func internalRemoteUsername(username string) bool {
	if !strings.HasPrefix(username, control.RolePrefix) {
		return false
	}
	for _, suffix := range []string{"_custodian", "_owner", "_reader", "_writer", "_read", "_write", "_observer"} {
		if strings.HasSuffix(username, suffix) {
			return true
		}
	}
	return strings.Contains(username, "_transfer_")
}

func certificatePool(encoded []byte) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	remaining := encoded
	count := 0
	for len(bytes.TrimSpace(remaining)) != 0 {
		remaining = bytes.TrimSpace(remaining)
		if !bytes.HasPrefix(remaining, []byte("-----BEGIN CERTIFICATE-----")) {
			return nil, InvalidRemoteTLS
		}
		block, rest := pem.Decode(remaining)
		if block == nil || block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			return nil, InvalidRemoteTLS
		}
		certificate, parseError := x509.ParseCertificate(block.Bytes)
		if parseError != nil || !certificate.IsCA {
			return nil, InvalidRemoteTLS
		}
		pool.AddCert(certificate)
		count++
		remaining = rest
	}
	if count == 0 {
		return nil, InvalidRemoteTLS
	}
	return pool, nil
}
