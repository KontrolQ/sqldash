package databases

import (
	"errors"

	"sqldash/pgtransfer"
)

func remoteFailureMessage(err error) string {
	for _, failure := range []error{
		pgtransfer.InvalidRemoteSource,
		pgtransfer.InvalidRemoteTLS,
		pgtransfer.RemoteSourceDenied,
		pgtransfer.RemoteConnectionFailed,
		pgtransfer.RemoteVersionUnsupported,
		pgtransfer.RemoteMetadataInvalid,
		pgtransfer.RemoteObjectsUnsupported,
		pgtransfer.RemoteExtensionsRequired,
		pgtransfer.UnsupportedPlatform,
		pgtransfer.InvalidIdentity,
		pgtransfer.InvalidStaging,
		pgtransfer.UnavailableClient,
		pgtransfer.UnsupportedClient,
		pgtransfer.AuthorizationDenied,
		pgtransfer.SnapshotFailed,
	} {
		if errors.Is(err, failure) {
			if failure == pgtransfer.RemoteConnectionFailed {
				return RemoteConnectionRefused
			}
			return failure.Error()
		}
	}
	return RemoteRefused
}
