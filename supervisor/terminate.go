package supervisor

import (
	"os/exec"
	"syscall"
	"time"
)

func stopAndWait(command *exec.Cmd, process Process, finished <-chan error) error {
	stopSignal := process.StopSignal
	if stopSignal == nil {
		stopSignal = syscall.SIGTERM
	}
	if signalError := command.Process.Signal(stopSignal); signalError != nil {
		_ = command.Process.Kill()
	}
	stopGrace := process.StopGrace
	if stopGrace <= 0 {
		stopGrace = ShutdownGrace
	}
	timer := time.NewTimer(stopGrace)
	defer timer.Stop()
	select {
	case waitError := <-finished:
		return waitError
	case <-timer.C:
		_ = command.Process.Kill()
		return <-finished
	}
}
