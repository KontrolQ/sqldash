package pgtransfer

import (
	"context"
	"os"

	"sqldash/models"
)

type RemoteSource struct {
	Host            string
	Port            int
	Database        string
	Username        string
	Password        string `json:"-"`
	SSLMode         string
	RootCertificate []byte
}

func (self RemoteSource) String() string {
	return RemoteSourceLabel
}

func (self RemoteSource) GoString() string {
	return self.String()
}

type RemoteExtension struct {
	Name    string
	Version string
	Schema  string
}

type RemoteManifest struct {
	ServerVersion int
	Database      string
	Extensions    []RemoteExtension
}

type RemoteRequirementsError struct {
	Requirements []RemoteExtension
}

func (self *RemoteRequirementsError) Error() string {
	return RemoteExtensionsRequired.Error()
}

func (self *RemoteRequirementsError) Unwrap() error {
	return RemoteExtensionsRequired
}

func InspectRemote(ctx context.Context, source RemoteSource) (*RemoteManifest, error) {
	return InspectRemoteWithOptions(ctx, source, Options{})
}

func InspectRemoteWithOptions(ctx context.Context, source RemoteSource, options Options) (*RemoteManifest, error) {
	if validationError := ValidateRemoteSource(source); validationError != nil {
		return nil, validationError
	}
	var manifest *RemoteManifest
	inspectionError := withJob(ctx, options, func(current *job) error {
		if clientError := current.verifyClient(DumpClient); clientError != nil {
			return clientError
		}
		endpoint, endpointError := current.prepareRemote(source)
		if endpointError != nil {
			return endpointError
		}
		return current.remoteSnapshot(source, endpoint, func(inspected *RemoteManifest, _ string, _ []string) error {
			manifest = inspected
			return current.authorize()
		})
	})
	if inspectionError != nil {
		return nil, inspectionError
	}
	return manifest, nil
}

func CloneRemote(ctx context.Context, source RemoteSource, target *models.Database) error {
	return CloneRemoteWithOptions(ctx, source, target, Options{})
}

func CloneRemoteWithOptions(ctx context.Context, source RemoteSource, target *models.Database, options Options) error {
	if validationError := ValidateRemoteSource(source); validationError != nil {
		return validationError
	}
	if validateTarget(target) != nil {
		return InvalidTarget
	}
	return withJob(ctx, options, func(current *job) error {
		if clientError := current.verifyClient(DumpClient); clientError != nil {
			return clientError
		}
		endpoint, endpointError := current.prepareRemote(source)
		if endpointError != nil {
			return endpointError
		}
		archive, createError := current.createFile("remote-clone")
		if createError != nil {
			return createError
		}
		defer archive.Close()
		dumpError := current.remoteSnapshot(source, endpoint, func(manifest *RemoteManifest, snapshot string, schemas []string) error {
			if requirementsError := current.remoteRequirements(target, manifest); requirementsError != nil {
				return requirementsError
			}
			return current.dumpRemote(source, endpoint, archive, snapshot, schemas)
		})
		if dumpError != nil {
			return dumpError
		}
		if cleanupError := current.removeRemoteCredentials(endpoint); cleanupError != nil {
			return cleanupError
		}
		return current.restore(target, archive)
	})
}

func (self *job) dumpRemote(source RemoteSource, endpoint *remoteEndpoint, output *os.File, snapshot string, schemas []string) error {
	arguments := []string{
		"--host=" + source.Host,
		"--port=" + endpoint.port,
		"--username=" + source.Username,
		"--dbname=" + source.Database,
		"--no-password",
	}
	arguments = dumpArguments(arguments, FormatCustom, snapshot, schemas)
	return self.runClientEnvironment(DumpClient, arguments, nil, output, self.maximumBytes, self.remoteEnvironment(source, endpoint))
}
