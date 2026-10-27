package pgtransfer

import (
	"context"
	"errors"
	"io"
	"os"
	"time"

	"sqldash/models"
	"sqldash/pgcluster/control"
)

type Options struct {
	Authorize    func(context.Context) error
	MaximumBytes int64
	Timeout      time.Duration
}

func Export(ctx context.Context, record *models.Database, writer io.Writer, format string) error {
	return ExportWithOptions(ctx, record, writer, format, Options{})
}

func ExportWithOptions(ctx context.Context, record *models.Database, writer io.Writer, format string, options Options) error {
	if validateSource(record) != nil {
		return InvalidDatabase
	}
	if writer == nil {
		return TransferFailed
	}
	format, formatError := normalizeFormat(format)
	if formatError != nil {
		return formatError
	}
	return withJob(ctx, options, func(current *job) error {
		output, createError := current.createFile("export")
		if createError != nil {
			return createError
		}
		defer output.Close()
		if dumpError := current.dump(record, output, format); dumpError != nil {
			return dumpError
		}
		if authorizeError := current.authorize(); authorizeError != nil {
			return authorizeError
		}
		if _, seekError := output.Seek(0, io.SeekStart); seekError != nil {
			return TransferFailed
		}
		return current.copy(writer, output)
	})
}

func Import(ctx context.Context, record *models.Database, sourcePath string) error {
	return ImportWithOptions(ctx, record, sourcePath, Options{})
}

func ImportWithOptions(ctx context.Context, record *models.Database, sourcePath string, options Options) error {
	if validateTarget(record) != nil {
		return InvalidTarget
	}
	return withJob(ctx, options, func(current *job) error {
		input, stageError := current.stage(sourcePath)
		if stageError != nil {
			return stageError
		}
		defer input.Close()
		return current.restore(record, input)
	})
}

func Clone(ctx context.Context, source, target *models.Database) error {
	return CloneWithOptions(ctx, source, target, Options{})
}

func CloneWithOptions(ctx context.Context, source, target *models.Database, options Options) error {
	if validateSource(source) != nil {
		return InvalidDatabase
	}
	if validateTarget(target) != nil || source.ID == target.ID || source.Name == target.Name || source.BackendOID == target.BackendOID {
		return InvalidTarget
	}
	return withJob(ctx, options, func(current *job) error {
		archive, createError := current.createFile("clone")
		if createError != nil {
			return createError
		}
		defer archive.Close()
		if dumpError := current.dump(source, archive, FormatCustom); dumpError != nil {
			return dumpError
		}
		return current.restore(target, archive)
	})
}

func normalizeFormat(format string) (string, error) {
	switch format {
	case FormatSQL, "plain":
		return FormatSQL, nil
	case FormatCustom, "archive":
		return FormatCustom, nil
	default:
		return "", InvalidFormat
	}
}

func validateRecord(record *models.Database) bool {
	return record != nil && record.ID != 0 && record.Kind() == models.EnginePostgres && record.Name != "" && record.BackendOID != 0 && record.ClusterID != ""
}

func validateSource(record *models.Database) error {
	if !validateRecord(record) || record.State != string(models.DatabaseReady) {
		return InvalidDatabase
	}
	return nil
}

func validateTarget(record *models.Database) error {
	if !validateRecord(record) || record.State != string(models.DatabaseApplying) {
		return InvalidTarget
	}
	return nil
}

func (self *job) withLogin(record *models.Database, mode control.SessionMode, operation func(*control.TransferCredential) error) (resultError error) {
	if authorizeError := self.authorize(); authorizeError != nil {
		return authorizeError
	}
	credential, loginError := control.TransferLogin(self.context, record, mode)
	if credential != nil {
		defer func() {
			cleanupError := self.cleanupLogin(record, credential, mode == control.Objects && resultError == nil)
			if cleanupError != nil {
				resultError = errors.Join(resultError, cleanupError)
			} else if resultError == nil {
				resultError = self.authorize()
			}
		}()
	}
	if loginError != nil || credential == nil {
		return self.safeError(LoginFailed)
	}
	if credentialError := self.writePassword(credential); credentialError != nil {
		return credentialError
	}
	if authorizeError := self.authorize(); authorizeError != nil {
		return authorizeError
	}
	if operationError := operation(credential); operationError != nil {
		return operationError
	}
	return self.authorize()
}

func (self *job) cleanupLogin(record *models.Database, credential *control.TransferCredential, success bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), CleanupTimeout)
	defer cancel()
	cleanupError := control.FinishTransfer(ctx, record, credential.Username, success)
	credential.Password = ""
	if cleanupError != nil {
		if success {
			control.FinishTransfer(ctx, record, credential.Username, false)
		}
		return CleanupFailed
	}
	return nil
}

func (self *job) dump(record *models.Database, output *os.File, format string) error {
	if clientError := self.verifyClient(DumpClient); clientError != nil {
		return clientError
	}
	return self.withLogin(record, control.ReadOnly, func(credential *control.TransferCredential) error {
		return self.withSnapshot(record, credential, func(snapshot string, schemas []string) error {
			arguments := dumpArguments(self.connectionArguments(credential), format, snapshot, schemas)
			return self.runClientEnvironment(DumpClient, arguments, nil, output, self.maximumBytes, self.privateEnvironment(true))
		})
	})
}

func (self *job) restore(record *models.Database, input *os.File) error {
	format, formatError := self.inputFormat(input)
	if formatError != nil {
		return formatError
	}
	client := RestoreClient
	if format == FormatSQL {
		client = SQLClient
	}
	if clientError := self.verifyClient(client); clientError != nil {
		return clientError
	}
	information, statError := input.Stat()
	if statError != nil {
		return InvalidSource
	}
	var source io.Reader = io.NewSectionReader(input, 0, information.Size())
	if format == FormatSQL {
		var readerError error
		source, readerError = restrictedSQL(input)
		if readerError != nil {
			return readerError
		}
	}
	return self.withLogin(record, control.Objects, func(credential *control.TransferCredential) error {
		if namespaceError := self.validatePublicNamespace(record, credential); namespaceError != nil {
			return namespaceError
		}
		arguments := self.connectionArguments(credential)
		if format == FormatCustom {
			list, listError := self.archiveList(input, information.Size())
			if listError != nil {
				return listError
			}
			arguments = append(arguments, "--no-owner", "--no-privileges", "--no-tablespaces", "--no-security-labels", "--exit-on-error", "--single-transaction", "--use-list="+list)
		} else {
			arguments = append(arguments, "-X", "--quiet", "--set=ON_ERROR_STOP=1", "--single-transaction", "--file=-")
		}
		return self.runClient(client, arguments, source, io.Discard, 0)
	})
}
