package secretstore

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

func privateDirectory(path string) error {
	if err := rejectSymlinkParents(path); err != nil {
		return err
	}
	if err := os.MkdirAll(path, DirectoryMode); err != nil {
		return err
	}
	if err := protectPath(path, true); err != nil {
		return err
	}
	return validateDirectory(path)
}

func rejectSymlinkParents(path string) error {
	for {
		information, err := os.Lstat(path)
		if err == nil && information.Mode()&os.ModeSymlink != 0 {
			return ErrUnsafePath
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		parent := filepath.Dir(path)
		if parent == path {
			return nil
		}
		path = parent
	}
}

func validateDirectory(path string) error {
	if err := rejectSymlinkParents(path); err != nil {
		return err
	}
	information, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !information.IsDir() || !privateMode(information.Mode(), true) {
		return ErrUnsafePath
	}
	return validateProtection(path, true)
}

func validatePrivateFile(path string) error {
	if err := rejectSymlinkParents(path); err != nil {
		return err
	}
	information, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !information.Mode().IsRegular() || !privateMode(information.Mode(), false) || information.Size() > maximumSecretBytes+1024 {
		return ErrUnsafePath
	}
	if err = validatePrivateLinks(path, information); err != nil {
		return err
	}
	return validateProtection(path, false)
}

func readPrivateFile(path string) ([]byte, error) {
	if err := validatePrivateFile(path); err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	information, err := file.Stat()
	if err != nil {
		return nil, err
	}
	pathInformation, err := os.Lstat(path)
	if err != nil || !os.SameFile(information, pathInformation) || !information.Mode().IsRegular() || !privateMode(information.Mode(), false) {
		return nil, ErrUnsafePath
	}
	return io.ReadAll(io.LimitReader(file, maximumSecretBytes+1024))
}

func atomicWrite(path string, contents []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".pending-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	defer temporary.Close()
	if err = protectPath(temporaryPath, false); err != nil {
		return err
	}
	if _, err = temporary.Write(contents); err != nil {
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
	return syncDirectory(filepath.Dir(path))
}
