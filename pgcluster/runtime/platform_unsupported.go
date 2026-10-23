//go:build !linux

package runtime

import (
	"os"
	"os/exec"

	"sqldash/supervisor"
)

func privateRegularFile(information os.FileInfo) bool { return information.Mode().IsRegular() }

func managedPlatform() error                                { return ErrUnsupportedPlatform }
func postgresCredentials() (*supervisor.Credentials, error) { return nil, ErrUnsupportedPlatform }
func configureCommand(command *exec.Cmd, credentials *supervisor.Credentials) error {
	return ErrUnsupportedPlatform
}
func setOwnership(path string, credentials *supervisor.Credentials) error {
	return ErrUnsupportedPlatform
}
func acquireClusterLock(path string) (*os.File, error) { return nil, ErrUnsupportedPlatform }
func releaseClusterLock(file *os.File)                 {}
func clusterRunning(path string) (bool, error)         { return false, ErrUnsupportedPlatform }
func fastShutdownSignal() os.Signal                    { return os.Interrupt }
func syncPathDirectory(path string) error              { return ErrUnsupportedPlatform }
