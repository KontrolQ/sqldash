package config

import (
	"errors"
	"math"
	"os"
	"path/filepath"
)

func verifyConfig() error {
	if postgresError := VerifyPostgres(); postgresError != nil {
		return postgresError
	}

	if redisError := VerifyRedis(); redisError != nil {
		return redisError
	}

	if Server.MaximumImageUploadBytes < 0 || Server.MaximumImageUploadBytes > math.MaxInt-1 {
		return errors.New(ImageUploadLimitInvalid)
	}

	if Server.MaximumWebSocketMessageBytes < 0 || Server.MaximumWebSocketMessageBytes > math.MaxInt64-1 || Sqld.QueryTimeout < 0 {
		return errors.New("WebSocket and libSQL query quotas must be zero or positive")
	}

	if Transfer.MaximumBytes < 0 || Transfer.MaximumBytes > int64(math.MaxInt-MultipartUploadOverhead) || Transfer.Timeout < 0 || Transfer.MaximumConcurrentJobs < 0 {
		return errors.New("transfer quotas must be zero or positive")
	}

	if Query.MaximumResultBytes < 0 || Query.MaximumResultBytes > math.MaxInt64-1 || Query.MaximumRows < 0 {
		return errors.New("query quotas must be zero or positive")
	}

	if History.Retention < 0 || History.MaximumPictures < 0 {
		return errors.New("history quotas must be zero or positive")
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
