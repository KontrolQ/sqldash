//go:build !windows

package redisconnections

import (
	"context"
	"errors"
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func safeFence(information os.FileInfo) bool {
	attributes, present := information.Sys().(*syscall.Stat_t)
	return present && attributes.Uid == uint32(os.Geteuid()) && attributes.Nlink == 1 && information.Mode().Perm()&0o077 == 0
}
func lockFence(ctx context.Context, file *os.File) error {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			return nil
		}
		if !errors.Is(err, unix.EWOULDBLOCK) && !errors.Is(err, unix.EAGAIN) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
func unlockFence(file *os.File) { _ = unix.Flock(int(file.Fd()), unix.LOCK_UN) }
