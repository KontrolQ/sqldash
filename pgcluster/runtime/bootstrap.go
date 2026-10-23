package runtime

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"sqldash/config"
	"sqldash/supervisor"
)

func verifyBinaries(ctx context.Context, credentials *supervisor.Credentials) error {
	for _, binaryName := range []string{"postgres", "initdb", "pg_controldata", "psql", "pg_dump", "pg_restore"} {
		output, err := runMetadataCommand(ctx, credentials, binaryName, "--version")
		if err != nil || !strings.HasPrefix(strings.TrimSpace(output), binaryName+" (PostgreSQL) 18.") {
			return errors.New("PostgreSQL 18 server and client binaries are required")
		}
	}
	return nil
}

func inspectDataDirectory() (bool, error) {
	dataPath := config.PostgresDataPath()
	if err := safePath(dataPath); err != nil {
		return false, err
	}
	information, err := os.Lstat(dataPath)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	if err != nil || !information.IsDir() {
		return false, ErrClusterDamaged
	}
	entries, err := os.ReadDir(dataPath)
	if err != nil {
		return false, err
	}
	if len(entries) == 0 {
		return true, nil
	}
	versionPath := filepath.Join(dataPath, "PG_VERSION")
	if err = regularFile(versionPath); err != nil {
		return false, ErrClusterDamaged
	}
	version, err := os.ReadFile(versionPath)
	if err != nil || strings.TrimSpace(string(version)) != MajorVersion {
		return false, ErrClusterDamaged
	}
	for _, relativePath := range []string{"global/pg_control", "postgresql.conf", "pg_hba.conf"} {
		path := filepath.Join(dataPath, filepath.FromSlash(relativePath))
		if err = regularFile(path); err != nil {
			return false, ErrClusterDamaged
		}
		information, err = os.Stat(path)
		if err != nil || information.Size() == 0 {
			return false, ErrClusterDamaged
		}
	}
	for _, relativePath := range []string{"base/1", "base/4", "base/5", "pg_wal"} {
		path := filepath.Join(dataPath, filepath.FromSlash(relativePath))
		if err = safePath(path); err != nil {
			return false, ErrClusterDamaged
		}
		information, err = os.Lstat(path)
		if err != nil || !information.IsDir() {
			return false, ErrClusterDamaged
		}
		if relativePath != "pg_wal" {
			entries, err = os.ReadDir(path)
			if err != nil || len(entries) == 0 {
				return false, ErrClusterDamaged
			}
		}
	}
	for _, signalName := range []string{"standby.signal", "recovery.signal"} {
		if _, err = os.Lstat(filepath.Join(dataPath, signalName)); !errors.Is(err, os.ErrNotExist) {
			return false, ErrClusterDamaged
		}
	}
	return false, nil
}

func initialize(ctx context.Context, credentials *supervisor.Credentials, passwordPath string) error {
	initializationContext, cancel := context.WithTimeout(ctx, config.Postgres.InitializationTimeout)
	defer cancel()
	command := exec.CommandContext(initializationContext, filepath.Join(config.Postgres.BinaryDirectory, "initdb"),
		"--pgdata="+config.PostgresDataPath(),
		"--username="+config.Postgres.AdminUsername,
		"--pwfile="+passwordPath,
		"--encoding=UTF8", "--locale=C.UTF-8", "--auth-local=scram-sha-256", "--auth-host=scram-sha-256", "--data-checksums")
	command.Dir = config.PostgresDirectory()
	command.Env = postgresEnvironment()
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if err := configureCommand(command, credentials); err != nil {
		return err
	}
	if err := command.Run(); err != nil {
		if initializationContext.Err() != nil {
			return initializationContext.Err()
		}
		return errors.New("PostgreSQL initialization failed; any partial data has been preserved")
	}
	return nil
}

func clusterIdentifier(ctx context.Context, credentials *supervisor.Credentials) (string, error) {
	identifier, state, err := readClusterControl(ctx, credentials)
	if err != nil {
		return "", err
	}
	switch state {
	case "shut down", "in production", "shutting down", "in crash recovery":
		return identifier, nil
	default:
		return "", ErrClusterDamaged
	}
}

func readClusterControl(ctx context.Context, credentials *supervisor.Credentials) (string, string, error) {
	output, err := runMetadataCommand(ctx, credentials, "pg_controldata", "--pgdata="+config.PostgresDataPath())
	if err != nil {
		return "", "", ErrClusterDamaged
	}
	var identifier, state string
	for _, line := range strings.Split(output, "\n") {
		name, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		switch strings.TrimSpace(name) {
		case "Database system identifier":
			identifier = strings.TrimSpace(value)
		case "Database cluster state":
			state = strings.TrimSpace(value)
		}
	}
	if numeric, numberError := strconv.ParseUint(identifier, 10, 64); numberError != nil || numeric == 0 || state == "" {
		return "", "", ErrClusterDamaged
	}
	return identifier, state, nil
}

func runMetadataCommand(ctx context.Context, credentials *supervisor.Credentials, name string, arguments ...string) (string, error) {
	commandContext, cancel := context.WithTimeout(ctx, config.Postgres.StartupTimeout)
	defer cancel()
	command := exec.CommandContext(commandContext, filepath.Join(config.Postgres.BinaryDirectory, name), arguments...)
	command.Dir = config.PostgresDirectory()
	command.Env = append(postgresEnvironment(), "LC_ALL=C")
	var output bytes.Buffer
	command.Stdout = &boundedOutput{buffer: &output, remaining: 64 * 1024}
	var diagnostics bytes.Buffer
	command.Stderr = &boundedOutput{buffer: &diagnostics, remaining: 64 * 1024}
	if err := configureCommand(command, credentials); err != nil {
		return "", err
	}
	if err := command.Run(); err != nil {
		return "", err
	}
	if diagnostics.Len() != 0 {
		return "", errors.New("PostgreSQL metadata validation reported a warning")
	}
	return output.String(), nil
}

type boundedOutput struct {
	buffer    *bytes.Buffer
	remaining int
}

func (self *boundedOutput) Write(contents []byte) (int, error) {
	if len(contents) > self.remaining {
		return 0, errors.New("PostgreSQL metadata exceeded its limit")
	}
	self.remaining -= len(contents)
	return self.buffer.Write(contents)
}

func postgresEnvironment() []string {
	return []string{"PATH=" + config.Postgres.BinaryDirectory + ":/usr/local/bin:/usr/bin:/bin", "LANG=C.UTF-8", "LC_ALL=C.UTF-8"}
}
