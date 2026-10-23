package mutation

import (
	"context"
	"errors"
	"os"
	"strconv"
	"sync/atomic"

	"sqldash/config"
)

type contextKey struct{}

type ownership struct {
	databaseID uint
	active     atomic.Bool
}

func With(ctx context.Context, databaseID uint, mutate func(context.Context) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if databaseID == 0 || mutate == nil {
		return errors.New("invalid credential mutation")
	}
	if held, ok := ctx.Value(contextKey{}).(*ownership); ok && held.databaseID == databaseID && held.active.Load() {
		return mutate(ctx)
	}
	path := config.StatePath() + ".credentials-" + strconv.FormatUint(uint64(databaseID), 10) + ".lock"
	if information, err := os.Lstat(path); err == nil && !information.Mode().IsRegular() {
		return errors.New("invalid credential mutation fence")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	pathInformation, pathError := os.Lstat(path)
	fileInformation, fileError := file.Stat()
	if pathError != nil || fileError != nil || !fileInformation.Mode().IsRegular() || !os.SameFile(pathInformation, fileInformation) {
		return errors.New("invalid credential mutation fence")
	}
	if err := lock(ctx, file); err != nil {
		return err
	}
	defer unlock(file)
	// The file stays in place so future processes lock the same inode.
	held := &ownership{databaseID: databaseID}
	held.active.Store(true)
	defer held.active.Store(false)
	return mutate(context.WithValue(ctx, contextKey{}, held))
}
