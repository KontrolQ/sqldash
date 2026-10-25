package redisconnections

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"

	"sqldash/config"
)

func fenced(ctx context.Context, databaseID uint, operation func() error) error {
	if databaseID == 0 || ctx.Err() != nil {
		return Conflict
	}
	path := config.StatePath() + ".redis-credentials-" + strconv.FormatUint(uint64(databaseID), 10) + ".lock"
	for checked := path; ; checked = filepath.Dir(checked) {
		information, err := os.Lstat(checked)
		if err == nil && information.Mode()&os.ModeSymlink != 0 {
			return Unavailable
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return Unavailable
		}
		if filepath.Dir(checked) == checked {
			break
		}
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return Unavailable
	}
	defer file.Close()
	pathInformation, pathError := os.Lstat(path)
	fileInformation, fileError := file.Stat()
	if pathError != nil || fileError != nil || !fileInformation.Mode().IsRegular() || !os.SameFile(pathInformation, fileInformation) || !safeFence(fileInformation) {
		return Unavailable
	}
	if err := lockFence(ctx, file); err != nil {
		return Conflict
	}
	defer unlockFence(file)
	return operation()
}
