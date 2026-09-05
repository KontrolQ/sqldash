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

	supervisor.Run(
		supervisor.Process{
			Name:      SqldProcessName,
			Path:      config.Sqld.BinaryPath,
			Arguments: sqldArguments(),
		},
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
