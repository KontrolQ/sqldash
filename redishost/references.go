package redishost

import (
	"context"

	"sqldash/models"
)

const referenceSecret = Profile("value-references")

func ReferenceKey(ctx context.Context, record *models.Database) ([]byte, error) {
	instance, err := Instance(ctx, record)
	if err != nil || instance.State != models.DatabaseReady {
		return nil, Unavailable
	}
	if _, err := readManifest(instance); err != nil {
		return nil, err
	}
	if err := checkQuarantine(instance); err != nil {
		return nil, err
	}
	return systemPassword(instance, referenceSecret, false)
}
