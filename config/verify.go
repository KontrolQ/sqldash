package config

import (
	"errors"
	"math"
	"os"
	"path/filepath"
)

func verifyConfig() error {
	if Server.MaximumImageUploadBytes <= 0 || Server.MaximumImageUploadBytes > math.MaxInt-MultipartUploadOverhead {
		return errors.New(ImageUploadLimitInvalid)
	}

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
