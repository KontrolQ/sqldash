package runtime

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"sqldash/config"
	"sqldash/supervisor"
)

func safePath(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	for {
		information, pathError := os.Lstat(absolute)
		if pathError == nil && information.Mode()&os.ModeSymlink != 0 {
			return errors.New("PostgreSQL storage must not contain symbolic links")
		}
		if pathError != nil && !errors.Is(pathError, os.ErrNotExist) {
			return pathError
		}
		parent := filepath.Dir(absolute)
		if parent == absolute {
			return nil
		}
		absolute = parent
	}
}

func regularFile(path string) error {
	if err := safePath(path); err != nil {
		return err
	}
	information, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !privateRegularFile(information) {
		return ErrClusterDamaged
	}
	return nil
}

func ensureDirectory(path string, credentials *supervisor.Credentials, mode os.FileMode) error {
	if err := safePath(path); err != nil {
		return err
	}
	if err := os.MkdirAll(path, mode); err != nil {
		return err
	}
	information, err := os.Lstat(path)
	if err != nil || !information.IsDir() {
		return ErrClusterDamaged
	}
	if err = setOwnership(path, credentials); err != nil {
		return err
	}
	if err = os.Chmod(path, mode); err != nil {
		return err
	}
	return syncPathDirectory(filepath.Dir(path))
}

func ownDataDirectory(path string, credentials *supervisor.Credentials) error {
	return filepath.WalkDir(path, func(entryPath string, entry fs.DirEntry, walkError error) error {
		if walkError != nil {
			return walkError
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return ErrClusterDamaged
		}
		information, err := entry.Info()
		if err != nil {
			return err
		}
		if !entry.IsDir() && !privateRegularFile(information) {
			return ErrClusterDamaged
		}
		if err = setOwnership(entryPath, credentials); err != nil {
			return err
		}
		mode := os.FileMode(0o600)
		if entry.IsDir() {
			mode = 0o700
		}
		return os.Chmod(entryPath, mode)
	})
}

func writePrivateFile(path string, contents []byte, credentials *supervisor.Credentials) error {
	if err := safePath(path); err != nil {
		return err
	}
	if err := regularFile(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".pending-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	defer temporary.Close()
	if _, err = temporary.Write(contents); err != nil {
		return err
	}
	if err = setOwnership(temporaryPath, credentials); err != nil {
		return err
	}
	if err = temporary.Chmod(0o600); err != nil {
		return err
	}
	if err = temporary.Sync(); err != nil {
		return err
	}
	if err = temporary.Close(); err != nil {
		return err
	}
	if err = os.Rename(temporaryPath, path); err != nil {
		return err
	}
	return syncPathDirectory(filepath.Dir(path))
}

func removeBootstrapPendingFiles() error {
	entries, err := os.ReadDir(config.PostgresConfigPath())
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".pending-") {
			if err = removePrivateFile(filepath.Join(config.PostgresConfigPath(), entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

func removePrivateFile(path string) error {
	if err := regularFile(path); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	return syncPathDirectory(filepath.Dir(path))
}
