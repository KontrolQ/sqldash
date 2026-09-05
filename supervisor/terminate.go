package supervisor

import (
	"os/exec"
	"syscall"
)

func terminate(command *exec.Cmd) {
	if signalError := command.Process.Signal(syscall.SIGTERM); signalError != nil {
		command.Process.Kill()
	}
}
