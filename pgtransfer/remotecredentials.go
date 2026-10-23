package pgtransfer

import (
	"os"
	"path/filepath"
)

func (self *job) prepareRemoteCredentials(source RemoteSource, endpoint *remoteEndpoint) error {
	if endpoint.sslMode == SSLVerifyFull || endpoint.sslMode == SSLVerifyCA {
		encoded := source.RootCertificate
		if len(encoded) == 0 {
			var readError error
			encoded, readError = os.ReadFile(SystemCertificatePath)
			if readError != nil || len(encoded) > MaximumCertificateBytes {
				return InvalidRemoteTLS
			}
		}
		pool, poolError := certificatePool(encoded)
		if poolError != nil {
			return poolError
		}
		certificate, createError := self.createFile("remote-ca")
		if createError != nil {
			return createError
		}
		_, writeError := certificate.Write(encoded)
		closeError := certificate.Close()
		if writeError != nil || closeError != nil {
			return InvalidStaging
		}
		endpoint.roots = pool
		endpoint.certificatePath = filepath.Join(self.directory, "remote-ca")
	}
	if passwordError := self.writePassfile("remote-pgpass", []string{source.Host, endpoint.port, source.Database, source.Username, source.Password}); passwordError != nil {
		return passwordError
	}
	endpoint.passwordPath = filepath.Join(self.directory, "remote-pgpass")
	return nil
}

func (self *job) removeRemoteCredentials(endpoint *remoteEndpoint) error {
	if removeError := self.root.Remove("remote-pgpass"); removeError != nil {
		return StagingCleanupFailed
	}
	if endpoint.certificatePath != "" {
		if removeError := self.root.Remove("remote-ca"); removeError != nil {
			return StagingCleanupFailed
		}
	}
	return nil
}
