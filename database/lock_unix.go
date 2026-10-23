//go:build !windows

package database

import (
	"context"
	"errors"
	"golang.org/x/sys/unix"
	"os"
	"time"
)

func lockSchema(parent context.Context, file *os.File) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		lockError := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if lockError == nil {
			return nil
		}
		if !errors.Is(lockError, unix.EWOULDBLOCK) {
			return lockError
		}
		select {
		case <-parent.Done():
			return parent.Err()
		case <-ticker.C:
		}
	}
}

func unlockSchema(file *os.File) {
	_ = unix.Flock(int(file.Fd()), unix.LOCK_UN)
}
