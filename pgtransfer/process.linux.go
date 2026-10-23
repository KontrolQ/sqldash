//go:build linux

package pgtransfer

import (
	"errors"
	"os"
	"os/exec"
	"syscall"

	"sqldash/supervisor"
)

func executionIdentity() error {
	if os.Geteuid() != 0 {
		return InvalidIdentity
	}
	return nil
}

func protectDirectory(path string) error {
	if chmodError := os.Chmod(path, PrivateDirectoryMode); chmodError != nil {
		return chmodError
	}
	return os.Chown(path, TransferUserID, TransferGroupID)
}

func protectFile(file *os.File) error {
	if chmodError := file.Chmod(PrivateFileMode); chmodError != nil {
		return chmodError
	}
	return file.Chown(TransferUserID, TransferGroupID)
}

func isolateCommand(command *exec.Cmd) error {
	if identityError := executionIdentity(); identityError != nil {
		return identityError
	}
	credentials := &supervisor.Credentials{UserID: TransferUserID, GroupID: TransferGroupID}
	if credentialError := credentials.Configure(command); credentialError != nil {
		return InvalidIdentity
	}
	command.SysProcAttr.Setpgid = true
	command.SysProcAttr.Pdeathsig = syscall.SIGKILL
	command.Cancel = func() error {
		if command.Process == nil {
			return os.ErrProcessDone
		}
		killError := syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		if errors.Is(killError, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return killError
	}
	return nil
}

func terminateCommand(command *exec.Cmd) {
	if command.Process != nil {
		syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	}
}
