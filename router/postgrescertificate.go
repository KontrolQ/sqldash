package router

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"github.com/gofiber/fiber/v3"
	"os"
	"path/filepath"
	"sqldash/config"
)

func publicPostgresCertificate(context fiber.Ctx) error {
	if !config.Postgres.Enabled || config.Postgres.CertificatePath != "" || config.Certificates.Automatic {
		return fiber.ErrNotFound
	}
	return publicNativeCertificate(context, config.PostgresTLSPath(), "sqldash-postgres-ca.crt")
}

func publicNativeCertificate(context fiber.Ctx, directory, filename string) error {
	path := filepath.Join(directory, "ca.crt")
	information, readError := os.Lstat(path)
	if readError != nil || !information.Mode().IsRegular() || information.Size() > 65536 {
		return fiber.ErrNotFound
	}
	contents, readError := os.ReadFile(path)
	if readError != nil {
		return fiber.ErrServiceUnavailable
	}
	block, remaining := pem.Decode(contents)
	if block == nil || block.Type != "CERTIFICATE" || len(remaining) != 0 {
		return fiber.ErrServiceUnavailable
	}
	certificate, parseError := x509.ParseCertificate(block.Bytes)
	if parseError != nil || !certificate.IsCA {
		return fiber.ErrServiceUnavailable
	}
	fingerprint := sha256.Sum256(certificate.Raw)
	context.Set("Content-Type", "application/x-pem-file")
	context.Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	context.Set("Cache-Control", "no-store")
	context.Set("X-Certificate-SHA256", hex.EncodeToString(fingerprint[:]))
	return context.Send(contents)
}
