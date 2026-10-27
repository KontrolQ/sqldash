package pgtransfer

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"sqldash/config"
	"sqldash/pgcluster/control"
	"sqldash/transferpolicy"
)

type job struct {
	context      context.Context
	options      Options
	maximumBytes int64
	directory    string
	root         *os.Root
	staging      *os.Root
	stagingPath  string
}

func normalizeOptions(options Options) (Options, error) {
	if options.MaximumBytes < 0 || options.Timeout < 0 {
		return options, InvalidOptions
	}
	options.MaximumBytes = transferpolicy.ByteLimit(options.MaximumBytes)
	if config.Transfer.Timeout > 0 && (options.Timeout == 0 || config.Transfer.Timeout < options.Timeout) {
		options.Timeout = config.Transfer.Timeout
	}
	return options, nil
}

func withJob(ctx context.Context, options Options, operation func(*job) error) (resultError error) {
	if ctx == nil {
		return InvalidOptions
	}
	var optionsError error
	options, optionsError = normalizeOptions(options)
	if optionsError != nil {
		return optionsError
	}
	if identityError := executionIdentity(); identityError != nil {
		return identityError
	}
	ctx, cancel := transferpolicy.Context(ctx, options.Timeout)
	defer cancel()
	release, capacityError := transferpolicy.Acquire(ctx)
	if capacityError != nil {
		return capacityError
	}
	defer release()
	current := &job{context: ctx, options: options, maximumBytes: options.MaximumBytes}
	if authorizeError := current.authorize(); authorizeError != nil {
		return authorizeError
	}
	if prepareError := current.prepare(); prepareError != nil {
		return prepareError
	}
	defer func() {
		current.root.Close()
		current.staging.Close()
		if removeError := os.RemoveAll(current.directory); removeError != nil {
			resultError = errors.Join(resultError, StagingCleanupFailed)
		}
	}()
	return operation(current)
}

func (self *job) prepare() error {
	stagingPath, pathError := filepath.Abs(config.PostgresStagingPath())
	if pathError != nil {
		return InvalidStaging
	}
	information, statError := os.Lstat(stagingPath)
	if statError != nil || !information.IsDir() || information.Mode()&os.ModeSymlink != 0 || information.Mode().Perm()&0o022 != 0 {
		return InvalidStaging
	}
	staging, stagingError := os.OpenRoot(stagingPath)
	if stagingError != nil {
		return InvalidStaging
	}
	directory, createError := os.MkdirTemp(stagingPath, "transfer-")
	if createError != nil {
		staging.Close()
		return InvalidStaging
	}
	if protectError := protectDirectory(directory); protectError != nil {
		os.RemoveAll(directory)
		staging.Close()
		return InvalidStaging
	}
	root, rootError := os.OpenRoot(directory)
	if rootError != nil {
		os.RemoveAll(directory)
		staging.Close()
		return InvalidStaging
	}
	self.directory = directory
	self.root = root
	self.staging = staging
	self.stagingPath = stagingPath
	return nil
}

func (self *job) createFile(name string) (*os.File, error) {
	file, createError := self.root.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, PrivateFileMode)
	if createError != nil {
		return nil, InvalidStaging
	}
	if protectError := protectFile(file); protectError != nil {
		file.Close()
		return nil, InvalidStaging
	}
	return file, nil
}

func (self *job) stage(sourcePath string) (*os.File, error) {
	absolutePath, pathError := filepath.Abs(sourcePath)
	if pathError != nil || sourcePath == "" {
		return nil, InvalidSource
	}
	relativePath, relativeError := filepath.Rel(self.stagingPath, absolutePath)
	if relativeError != nil || !filepath.IsLocal(relativePath) || relativePath == "." {
		return nil, InvalidSource
	}
	information, statError := self.staging.Lstat(relativePath)
	if statError != nil || !information.Mode().IsRegular() || information.Size() == 0 {
		return nil, InvalidSource
	}
	if self.maximumBytes > 0 && information.Size() > self.maximumBytes {
		return nil, SizeExceeded
	}
	source, openError := self.staging.Open(relativePath)
	if openError != nil {
		return nil, InvalidSource
	}
	defer source.Close()
	openedInformation, openedError := source.Stat()
	if openedError != nil || !openedInformation.Mode().IsRegular() || !os.SameFile(information, openedInformation) {
		return nil, InvalidSource
	}
	input, createError := self.createFile("input")
	if createError != nil {
		return nil, createError
	}
	if copyError := self.copy(input, source); copyError != nil {
		input.Close()
		return nil, copyError
	}
	return input, nil
}

func (self *job) authorize() error {
	if contextError := self.context.Err(); contextError != nil {
		return contextError
	}
	if self.options.Authorize != nil {
		if authorizeError := self.options.Authorize(self.context); authorizeError != nil {
			return self.safeError(AuthorizationDenied)
		}
	}
	return nil
}

func (self *job) safeError(fallback error) error {
	if contextError := self.context.Err(); contextError != nil {
		return contextError
	}
	return fallback
}

func (self *job) copy(destination io.Writer, source io.Reader) error {
	bounded := &boundedWriter{destination: destination, maximumBytes: self.maximumBytes, limitError: SizeExceeded}
	_, copyError := io.CopyBuffer(bounded, &contextReader{context: self.context, source: source}, make([]byte, CopyBufferBytes))
	if copyError != nil {
		if bounded.exceeded {
			return SizeExceeded
		}
		return self.safeError(TransferFailed)
	}
	return nil
}

func (self *job) writePassword(credential *control.TransferCredential) error {
	if !validCredential(credential) {
		return InvalidAddress
	}
	host, port, _ := net.SplitHostPort(credential.Address)
	return self.writePassfile("pgpass", []string{host, port, credential.Database, credential.Username, credential.Password})
}

func (self *job) writePassfile(name string, fields []string) error {
	file, openError := self.root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, PrivateFileMode)
	if openError != nil {
		return InvalidStaging
	}
	defer file.Close()
	if protectError := protectFile(file); protectError != nil {
		return InvalidStaging
	}
	for index, field := range fields {
		fields[index] = strings.NewReplacer("\\", "\\\\", ":", "\\:").Replace(field)
	}
	encoded := []byte(strings.Join(fields, ":") + "\n")
	defer clear(encoded)
	if _, writeError := file.Write(encoded); writeError != nil {
		return InvalidStaging
	}
	return nil
}

func validCredential(credential *control.TransferCredential) bool {
	if credential == nil || credential.Username == "" || credential.Password == "" || credential.Database == "" || len(credential.Database) > 63 || !utf8.ValidString(credential.Database) || strings.Contains(credential.Database, "=") || strings.HasPrefix(credential.Database, "postgresql://") || strings.HasPrefix(credential.Database, "postgres://") {
		return false
	}
	for _, value := range []string{credential.Username, credential.Password, credential.Database, credential.Address} {
		if strings.ContainsAny(value, "\r\n\x00") {
			return false
		}
	}
	host, portText, addressError := net.SplitHostPort(credential.Address)
	port, portError := strconv.ParseUint(portText, 10, 16)
	address := net.ParseIP(host)
	if addressError != nil || portError != nil || port == 0 || address == nil || !address.IsLoopback() || credential.Address != config.Postgres.Address {
		return false
	}
	return !config.Postgres.Managed || host == "127.0.0.1" && port == 55432
}

func (self *job) connectionArguments(credential *control.TransferCredential) []string {
	host, port, _ := net.SplitHostPort(credential.Address)
	return []string{"--host=" + host, "--port=" + port, "--username=" + credential.Username, "--dbname=" + credential.Database, "--no-password"}
}
