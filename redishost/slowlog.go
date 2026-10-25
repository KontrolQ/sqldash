package redishost

import (
	"context"
	"sqldash/models"
)

func ResetSlowLog(ctx context.Context, record *models.Database) error {
	instance, err := Instance(ctx, record)
	if err != nil || instance.State != models.DatabaseReady {
		return Unavailable
	}
	control, err := privateClient(ctx, record, instance, systemProfile)
	if err != nil {
		return Unavailable
	}
	defer control.Close()
	if err := control.Do(ctx, "SLOWLOG", "RESET").Err(); err != nil {
		return Unavailable
	}
	return nil
}
