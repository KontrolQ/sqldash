package redishost

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"sqldash/config"
)

func rejectSymlinks(path string) error {
	for {
		information, err := os.Lstat(path)
		if err == nil && information.Mode()&os.ModeSymlink != 0 {
			return UnsafePath
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return UnsafePath
		}
		parent := filepath.Dir(path)
		if parent == path {
			return nil
		}
		path = parent
	}
}

func ownedDirectory(path string, mode os.FileMode, userID, groupID int) error {
	if err := rejectSymlinks(path); err != nil {
		return err
	}
	err := os.Mkdir(path, mode)
	created := err == nil
	if err != nil && !errors.Is(err, os.ErrExist) {
		return Unavailable
	}
	information, err := os.Lstat(path)
	if err != nil || !information.IsDir() || !created && !verifyOwner(information, userID) {
		return UnsafePath
	}
	if created {
		if err := os.Chown(path, userID, groupID); err != nil {
			return Unavailable
		}
	}
	if err := os.Chmod(path, mode); err != nil {
		return Unavailable
	}
	return nil
}

func ownedFile(path string, contents []byte, mode os.FileMode, groupID int) error {
	if err := rejectSymlinks(path); err != nil {
		return err
	}
	if information, err := os.Lstat(path); err == nil {
		if !information.Mode().IsRegular() || !verifyOwner(information, 0) {
			return UnsafePath
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return UnsafePath
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".pending-")
	if err != nil {
		return Unavailable
	}
	defer temporary.Close()
	defer os.Remove(temporary.Name())
	if err := temporary.Chown(0, groupID); err != nil {
		return Unavailable
	}
	if err := temporary.Chmod(mode); err != nil {
		return Unavailable
	}
	if _, err := temporary.Write(contents); err != nil {
		return Unavailable
	}
	if err := temporary.Sync(); err != nil {
		return Unavailable
	}
	if err := temporary.Close(); err != nil {
		return Unavailable
	}
	if err := os.Rename(temporary.Name(), path); err != nil {
		return Unavailable
	}
	directory, err := os.Open(filepath.Dir(path))
	if err != nil {
		return Unavailable
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return Unavailable
	}
	return nil
}

func readOwnedFile(path string) ([]byte, error) {
	if err := rejectSymlinks(path); err != nil {
		return nil, err
	}
	information, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !information.Mode().IsRegular() || !verifyOwner(information, 0) || information.Mode().Perm()&0o077 != 0 || information.Size() > maximumManifestBytes {
		return nil, UnsafePath
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, Unavailable
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(information, opened) {
		return nil, UnsafePath
	}
	return io.ReadAll(io.LimitReader(file, maximumManifestBytes+1))
}

func prepareDirectories() error {
	for _, entry := range []struct {
		path string
		mode os.FileMode
	}{
		{config.RedisDirectory(), 0o711},
		{stateDirectory(), 0o700},
		{filepath.Join(config.RedisDirectory(), "config"), 0o711},
		{filepath.Join(config.RedisDirectory(), "sockets"), 0o711},
		{config.RedisInstancesPath(), 0o711},
		{config.RedisRuntimePath(), 0o700},
		{config.RedisStagingPath(), 0o700},
		{filepath.Join(config.RedisDirectory(), "logs"), 0o700},
	} {
		if err := ownedDirectory(entry.path, entry.mode, 0, 0); err != nil {
			return err
		}
	}
	return nil
}
