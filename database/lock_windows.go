package database

import (
	"context"
	"errors"
	"golang.org/x/sys/windows"
	"os"
	"time"
)

func lockSchema(parent context.Context, file *os.File) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		overlapped := new(windows.Overlapped)
		lockError := windows.LockFileEx(windows.Handle(file.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, overlapped)
		if lockError == nil {
			return nil
		}
		if !errors.Is(lockError, windows.ERROR_LOCK_VIOLATION) {
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
	_ = windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, new(windows.Overlapped))
}
