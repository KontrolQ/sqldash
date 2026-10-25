package redishost

import (
	"context"
	"io/fs"
	"math"
	"os"

	"sqldash/models"
)

func StorageBytes(ctx context.Context, record *models.Database) (int64, error) {
	instance, err := Instance(ctx, record)
	if err != nil || instance.State != models.DatabaseReady {
		return 0, Unavailable
	}
	if _, err := readManifest(instance); err != nil {
		return 0, err
	}
	path := dataDirectory(record.ID)
	if err := rejectSymlinks(path); err != nil {
		return 0, err
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return 0, Unavailable
	}
	defer root.Close()
	userID, _, err := instanceCredentials(record.ID)
	if err != nil {
		return 0, err
	}
	var total int64
	visited := 0
	err = fs.WalkDir(root.FS(), ".", func(name string, entry fs.DirEntry, visitError error) error {
		if visitError != nil {
			return Unavailable
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		visited++
		if visited > 10000 || entry.Type()&os.ModeSymlink != 0 {
			return UnsafePath
		}
		information, err := entry.Info()
		if err != nil || !verifyOwner(information, userID) {
			return UnsafePath
		}
		if information.IsDir() {
			return nil
		}
		if !information.Mode().IsRegular() || information.Size() < 0 || total > math.MaxInt64-information.Size() {
			return UnsafePath
		}
		total += information.Size()
		return nil
	})
	if err != nil {
		return 0, err
	}
	return total, nil
}
