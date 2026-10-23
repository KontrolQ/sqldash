package supervisor

import (
	"errors"
	"os/exec"
)

func configureCredentials(command *exec.Cmd, credentials *Credentials) error {
	if credentials != nil {
		return errors.New("per-child Unix credentials are unsupported on Windows")
	}
	return nil
}
