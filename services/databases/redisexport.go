package databases

import (
	"context"
	"errors"
	"io"
	"os"

	"sqldash/config"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/redistransfer"
)

func WriteRedis(ctx context.Context, name string, writer io.Writer) error {
	fresh, record, err := authorizeExport(ctx, name)
	if err != nil {
		return err
	}
	if record.Kind() != models.EngineRedis {
		return errors.New(EngineRefused)
	}
	return redistransfer.ExportWithOptions(fresh, record, writer, redistransfer.Options{Authorize: func(step context.Context) error {
		_, current, err := Authorize(step, name, permissions.DatabaseExport, true)
		if err != nil {
			return err
		}
		if current.ID != record.ID || current.ClusterID != record.ClusterID {
			return errors.New(DatabaseMissing)
		}
		return nil
	}})
}

func RedisExport(ctx context.Context, name string) (io.ReadCloser, error) {
	return stagedNativeExport(ctx, name, models.EngineRedis, config.RedisStagingPath(), WriteRedis)
}

func stagedNativeExport(ctx context.Context, name, engine, directory string, write func(context.Context, string, io.Writer) error) (io.ReadCloser, error) {
	fresh, record, authorityError := authorizeExport(ctx, name)
	if authorityError != nil {
		return nil, authorityError
	}
	if record.Kind() != engine {
		return nil, errors.New(EngineRefused)
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	handle, err := os.CreateTemp(directory, "download-*")
	if err != nil {
		return nil, err
	}
	held := &stagedExport{File: handle, path: handle.Name(), ctx: fresh, name: record.Name, databaseID: record.ID}
	if err := handle.Chmod(0600); err != nil {
		held.Close()
		return nil, err
	}
	if err := write(fresh, record.Name, handle); err != nil {
		held.Close()
		return nil, err
	}
	if _, err := handle.Seek(0, io.SeekStart); err != nil {
		held.Close()
		return nil, err
	}
	return held, nil
}
