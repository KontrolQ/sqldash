//go:build !windows

package supervisor

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

func configureCredentials(command *exec.Cmd, credentials *Credentials) error {
	if credentials == nil {
		return nil
	}
	if credentials.UserID == 0 {
		return errors.New("a supervised child cannot use root credentials")
	}
	if os.Geteuid() != 0 {
		if uint32(os.Geteuid()) != credentials.UserID || uint32(os.Getegid()) != credentials.GroupID {
			return errors.New("the process cannot switch to the requested credentials")
		}
		return nil
	}
	command.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{
		Uid:    credentials.UserID,
		Gid:    credentials.GroupID,
		Groups: []uint32{},
	}}
	return nil
}
