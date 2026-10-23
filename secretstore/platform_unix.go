//go:build !windows

package secretstore

import (
	"os"
	"syscall"
)

func lockFile(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
}

func unlockFile(file *os.File) {
	_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}

func protectPath(path string, directory bool) error {
	mode := os.FileMode(FileMode)
	if directory {
		mode = DirectoryMode
	}
	return os.Chmod(path, mode)
}

func privateMode(mode os.FileMode, directory bool) bool {
	if directory {
		return mode.Perm() == DirectoryMode
	}
	return mode.Perm() == FileMode
}

func validatePrivateLinks(path string, information os.FileInfo) error {
	status, valid := information.Sys().(*syscall.Stat_t)
	if !valid || status.Nlink != 1 {
		return ErrUnsafePath
	}
	return nil
}

func validateProtection(path string, directory bool) error {
	information, err := os.Lstat(path)
	if err != nil {
		return err
	}
	status, valid := information.Sys().(*syscall.Stat_t)
	if !valid || status.Uid != uint32(os.Geteuid()) {
		return ErrUnsafePath
	}
	return nil
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
