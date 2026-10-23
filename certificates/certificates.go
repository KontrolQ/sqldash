package certificates

import (
	"context"
	"crypto/tls"

	"sqldash/config"
	"sqldash/utils/logger"

	"github.com/caddyserver/certmagic"
	"github.com/libdns/cloudflare"
)

func Wanted() bool {
	return config.Certificates.Automatic
}

func Names() []string {
	return []string{config.Server.Domain, Wildcard + config.Server.Domain}
}

func authority() string {
	if config.Certificates.Staging {
		return certmagic.LetsEncryptStagingCA
	}

	return certmagic.LetsEncryptProductionCA
}

func Settings() (*tls.Config, error) {
	return managedSettings(shutdownAware(), config.CertificatesPath(), Names())
}

func NativeSettings(parent context.Context, host string, storage string) (*tls.Config, error) {
	return managedSettings(parent, storage, []string{host})
}

func managedSettings(parent context.Context, storage string, names []string) (*tls.Config, error) {
	holder := certmagic.NewDefault()
	holder.Storage = &certmagic.FileStorage{Path: storage}
	holder.Logger = quiet()

	issuer := certmagic.NewACMEIssuer(holder, certmagic.ACMEIssuer{
		CA:     authority(),
		Email:  config.Certificates.Email,
		Agreed: true,
		DNS01Solver: &certmagic.DNS01Solver{
			DNSManager: certmagic.DNSManager{
				DNSProvider: &cloudflare.Provider{APIToken: config.Certificates.CloudflareAPIToken},
			},
		},
		Logger: quiet(),
	})

	holder.Issuers = []certmagic.Issuer{issuer}

	logger.Infof(LogPrefix, ObtainingLog, config.Server.Domain)

	if manageError := holder.ManageSync(parent, names); manageError != nil {
		return nil, manageError
	}

	logger.Successf(LogPrefix, ReadyLog, config.Server.Domain)

	settings := holder.TLSConfig()
	settings.NextProtos = append([]string{HTTP11}, settings.NextProtos...)
	settings.MinVersion = tls.VersionTLS12

	return settings, nil
}
