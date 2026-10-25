package redishost

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"sqldash/config"
)

func verifyRuntime(ctx context.Context) error {
	if !config.Redis.Enabled || !managedSupported() {
		return Unavailable
	}
	binary, err := exec.LookPath(config.Redis.BinaryPath)
	if err != nil || rejectSymlinks(binary) != nil {
		return Unavailable
	}
	information, err := os.Stat(binary)
	if err != nil || !information.Mode().IsRegular() || !verifyOwner(information, 0) || information.Mode().Perm()&0o022 != 0 {
		return UnsafePath
	}
	probe, cancel := context.WithTimeout(ctx, controlTimeout)
	defer cancel()
	command := exec.CommandContext(probe, binary, "--version")
	command.Env = []string{"PATH=/usr/local/bin:/usr/bin:/bin", "LANG=C.UTF-8", "HOME=/nonexistent"}
	output, err := command.Output()
	if err != nil || !bytes.HasPrefix(bytes.TrimSpace(output), []byte("Redis server v=8.")) {
		return Unavailable
	}
	for _, name := range moduleFiles {
		path := filepath.Join(config.Redis.ModulesDirectory, name)
		if err := rejectSymlinks(path); err != nil {
			return err
		}
		information, err := os.Stat(path)
		if err != nil || !information.Mode().IsRegular() || !verifyOwner(information, 0) || information.Mode().Perm()&0o022 != 0 {
			return Unavailable
		}
	}
	return nil
}
