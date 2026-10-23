package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sqldash/config"
	"sqldash/pgcluster/control"
	postgresRuntime "sqldash/pgcluster/runtime"
	"sqldash/supervisor"
	"sqldash/utils/logger"
	"syscall"
	"time"
)

func supervise() {
	executable, executableError := os.Executable()
	if executableError != nil {
		logger.Fatalf(LogPrefix, ExecutableLog, executableError)
	}
	parent, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	processes := make([]supervisor.Process, 0, 5)
	if config.Sqld.Managed {
		processes = append(processes, supervisor.Process{Name: SqldProcessName, Path: config.Sqld.BinaryPath, Arguments: sqldArguments(), ShutdownGroup: 100})
	} else {
		logger.Infof(LogPrefix, UnmanagedSqldLog, config.Sqld.Address)
	}
	var cluster *postgresRuntime.Runtime
	if config.Postgres.Enabled && config.Postgres.Managed {
		cluster = postgresRuntime.New()
		process, processError := cluster.Process(control.Ready)
		if processError != nil {
			logger.Fatalf(LogPrefix, "Could not configure PostgreSQL supervision: %v", processError)
		}
		prepare := process.Prepare
		process.Prepare = func(parent context.Context) error {
			if _, fileError := os.Lstat(postgresUnsafePath()); fileError == nil {
				return errors.New("PostgreSQL is blocked pending enforcement recovery")
			} else if !errors.Is(fileError, os.ErrNotExist) {
				return fileError
			}
			return prepare(parent)
		}
		processes = append(processes, process)
	}
	verified := []string{"SQLDASH_SCHEMA_VERIFIED=1"}
	processes = append(processes, supervisor.Process{Name: ProxyProcessName, Path: executable, Arguments: []string{ModeProxy}, Environment: verified, ShutdownGroup: 0}, supervisor.Process{Name: WebProcessName, Path: executable, Arguments: []string{ModeWeb}, Environment: verified, ShutdownGroup: 0})
	if config.Postgres.Enabled && config.Postgres.GatewayEnabled {
		processes = append(processes, supervisor.Process{Name: ModePostgresGateway, Path: executable, Arguments: []string{ModePostgresGateway}, Environment: verified, ShutdownGroup: 0})
	}
	manager := supervisor.New(processes...)
	if cluster != nil {
		go watchPostgresEnforcement(parent, manager, cluster)
		defer func() {
			if closeError := cluster.Close(); closeError != nil {
				logger.Errorf(LogPrefix, "PostgreSQL cluster closure could not be confirmed: %v", closeError)
			}
		}()
	}
	if runError := manager.Run(parent); runError != nil {
		logger.Errorf(LogPrefix, "Process supervision stopped: %v", runError)
	}
}

func watchPostgresEnforcement(parent context.Context, manager *supervisor.Supervisor, cluster *postgresRuntime.Runtime) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-parent.Done():
			return
		case <-ticker.C:
			if _, fileError := os.Lstat(postgresUnsafePath()); errors.Is(fileError, os.ErrNotExist) {
				continue
			}
			if stopError := manager.Stop(postgresRuntime.ProcessName); stopError != nil {
				logger.Errorf(LogPrefix, "PostgreSQL enforcement shutdown failed: %v", stopError)
				return
			}
			confirmation, cancel := context.WithTimeout(context.Background(), config.Postgres.ShutdownTimeout)
			shutdownError := cluster.StoppedSafe(confirmation)
			cancel()
			if shutdownError != nil {
				logger.Errorf(LogPrefix, "PostgreSQL enforcement shutdown is unconfirmed: %v", shutdownError)
			} else {
				logger.Warnf(LogPrefix, "PostgreSQL stopped after a session enforcement failure")
			}
			return
		}
	}
}

func sqldArguments() []string {
	return []string{"--db-path", config.DatabasesPath(), "--http-listen-addr", config.Sqld.Address, "--admin-listen-addr", config.Sqld.AdminAddress, "--enable-namespaces", "--no-welcome"}
}
