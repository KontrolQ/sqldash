package config

import (
	"errors"
	"os"
	"path/filepath"
)

func verifyConfig() error {
	if Server.Domain == "" {
		return errors.New(DomainMissing)
	}

	if Certificates.Automatic {
		if Certificates.CloudflareAPIToken == "" {
			return errors.New(CertificateTokenMissing)
		}

		if Certificates.Email == "" {
			return errors.New(CertificateEmailMissing)
		}
	}

	absolute, resolveError := filepath.Abs(Data.Directory)
	if resolveError != nil {
		return errors.New(DataDirectoryUnreachable)
	}

	Data.Directory = absolute

	for _, directory := range []string{DatabasesPath(), ImportsPath(), CertificatesPath()} {
		if makeError := os.MkdirAll(directory, DirectoryMode); makeError != nil {
			return errors.New(DataDirectoryUnreachable)
		}
	}

	return nil
}
