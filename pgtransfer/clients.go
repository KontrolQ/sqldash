package pgtransfer

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"sqldash/config"
)

func (self *job) verifyClient(client string) error {
	output := &bytes.Buffer{}
	if clientError := self.runClient(client, []string{"--version"}, nil, output, MaximumVersionBytes); clientError != nil {
		return clientError
	}
	prefix := client + " (PostgreSQL) 18."
	version := strings.TrimSpace(output.String())
	if !strings.HasPrefix(version, prefix) || len(version) == len(prefix) || version[len(prefix)] < '0' || version[len(prefix)] > '9' {
		return UnsupportedClient
	}
	return nil
}

func (self *job) runClient(client string, arguments []string, input io.Reader, output io.Writer, maximumBytes int64) error {
	return self.runClientEnvironment(client, arguments, input, output, maximumBytes, self.environment())
}

func (self *job) runClientEnvironment(client string, arguments []string, input io.Reader, output io.Writer, maximumBytes int64, environment []string) error {
	if authorizeError := self.authorize(); authorizeError != nil {
		return authorizeError
	}
	if client != DumpClient && client != RestoreClient && client != SQLClient || !filepath.IsAbs(config.Postgres.BinaryDirectory) {
		return UnavailableClient
	}
	path := filepath.Join(config.Postgres.BinaryDirectory, client)
	information, statError := os.Stat(path)
	if statError != nil || !information.Mode().IsRegular() || information.Mode().Perm()&0o022 != 0 || information.Mode()&(os.ModeSetuid|os.ModeSetgid) != 0 || information.Mode().Perm()&0o111 == 0 {
		return UnavailableClient
	}
	ctx, cancel := context.WithCancel(self.context)
	defer cancel()
	command := exec.CommandContext(ctx, path, arguments...)
	command.Dir = self.directory
	command.Env = environment
	command.WaitDelay = ClientWaitDelay
	command.Stdin = input
	limitError := OutputExceeded
	if client == DumpClient && len(arguments) != 1 {
		limitError = SizeExceeded
	}
	bounded := &boundedWriter{destination: output, maximumBytes: maximumBytes, limitError: limitError, cancel: cancel}
	diagnostics := &diagnosticBuffer{}
	defer diagnostics.clear()
	command.Stdout = bounded
	command.Stderr = diagnostics
	if isolationError := isolateCommand(command); isolationError != nil {
		return isolationError
	}
	defer terminateCommand(command)
	if runError := command.Run(); runError != nil {
		if bounded.exceeded {
			return limitError
		}
		if contextError := self.context.Err(); contextError != nil {
			return contextError
		}
		return ClientFailed
	}
	return nil
}

func (self *job) environment() []string {
	return self.privateEnvironment(false)
}

func (self *job) privateEnvironment(readOnly bool) []string {
	attributes := "read-write"
	if readOnly {
		attributes = "any"
	}
	return append(self.processEnvironment(readOnly),
		"PGHOST=127.0.0.1", "PGPORT=55432",
		"PGPASSFILE="+filepath.Join(self.directory, "pgpass"),
		"PGSSLMODE=disable", "PGTARGETSESSIONATTRS="+attributes,
	)
}

func (self *job) processEnvironment(readOnly bool) []string {
	return self.processEnvironmentVersion(readOnly, 180000)
}

func (self *job) processEnvironmentVersion(readOnly bool, version int) []string {
	settings := "-c statement_timeout=" + strconv.FormatInt(self.options.Timeout.Milliseconds(), 10)
	if version >= 90300 {
		settings += " -c lock_timeout=5000"
	}
	if version >= 90600 {
		settings += " -c idle_in_transaction_session_timeout=" + strconv.FormatInt(self.options.Timeout.Milliseconds(), 10)
	}
	if readOnly {
		settings += " -c default_transaction_read_only=on"
	}
	return []string{
		"HOME=" + self.directory,
		"TMPDIR=" + self.directory,
		"PATH=" + config.Postgres.BinaryDirectory,
		"LANG=C", "LC_ALL=C", "PGCONNECT_TIMEOUT=10",
		"PGCLIENTENCODING=UTF8", "PGAPPNAME=sqldash-transfer",
		"PGGSSENCMODE=disable", "PGSSLNEGOTIATION=postgres",
		"PGOPTIONS=" + settings,
	}
}
