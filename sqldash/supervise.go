package main

import (
	"os"

	"sqldash/config"
	"sqldash/supervisor"
	"sqldash/utils/logger"
)

func supervise() {
	executable, executableError := os.Executable()
	if executableError != nil {
		logger.Fatalf(LogPrefix, ExecutableLog, executableError)
	}

	processes := make([]supervisor.Process, 0, 3)

	if config.Sqld.Managed {
		processes = append(processes, supervisor.Process{
			Name:      SqldProcessName,
			Path:      config.Sqld.BinaryPath,
			Arguments: sqldArguments(),
		})
	} else {
		logger.Infof(LogPrefix, UnmanagedSqldLog, config.Sqld.Address)
	}

	processes = append(processes,
		supervisor.Process{
			Name:      ProxyProcessName,
			Path:      executable,
			Arguments: []string{ModeProxy},
		},
		supervisor.Process{
			Name:      WebProcessName,
			Path:      executable,
			Arguments: []string{ModeWeb},
		},
	)

	supervisor.Run(processes...)
}

func sqldArguments() []string {
	return []string{
		"--db-path", config.DatabasesPath(),
		"--http-listen-addr", config.Sqld.Address,
		"--admin-listen-addr", config.Sqld.AdminAddress,
		"--enable-namespaces",
		"--no-welcome",
	}
}
