package databases

import (
	"context"
	"errors"
	"os"

	"sqldash/database"
	"sqldash/engines"
	"sqldash/models"
	"sqldash/proxy"
	"sqldash/redishost"
	"sqldash/redistransfer"
	"sqldash/services/redisconnections"
	"sqldash/utils/logger"
)

func applyRedisOperation(ctx context.Context, record *models.Database, operation *models.DatabaseOperation, backend engines.Backend, payload operationPayload) error {
	fresh, source, err := operationAuthority(ctx, record, operation)
	if err != nil {
		return err
	}
	if operation.Kind == "delete" {
		return finishDelete(fresh, record, operation, backend)
	}
	if operation.Kind != "create" && operation.Kind != "import" && operation.Kind != "clone" || record.State == models.DatabaseDeleting {
		return errors.New(EngineRefused)
	}
	if payload.Phase == "" || payload.Phase == "creating" {
		payload.Phase = "creating"
		if err := savePhase(ctx, operation, payload); err != nil {
			return err
		}
		if err := backend.Create(fresh, record); err != nil {
			return err
		}
		record.State = models.DatabaseApplying
		if err := database.DB.WithContext(ctx).Model(record).Updates(map[string]any{"cluster_id": record.ClusterID, "backend_oid": 0, "state": record.State}).Error; err != nil {
			return err
		}
		payload.Phase = "applying"
		if err := savePhase(ctx, operation, payload); err != nil {
			return err
		}
	}
	if payload.Phase == "applying" {
		instance, err := redishost.Instance(fresh, record)
		if err != nil || instance.Published || instance.Identity != record.ClusterID {
			return errors.New(EngineRefused)
		}
		options := redistransfer.Options{Timeout: operationTimeout, Authorize: func(step context.Context) error { _, _, err := operationAuthority(step, record, operation); return err }}
		switch operation.Kind {
		case "create":
			err = backend.Create(fresh, record)
		case "import":
			err = redistransfer.ImportWithOptions(fresh, record, payload.Path, options)
		case "clone":
			err = redistransfer.CloneWithOptions(fresh, source, record, options)
		}
		if err != nil {
			return err
		}
		payload.Phase = "verify"
		if err := savePhase(ctx, operation, payload); err != nil {
			return err
		}
	}
	if payload.Phase != "verify" {
		return errors.New(EngineRefused)
	}
	fresh, _, err = operationAuthority(ctx, record, operation)
	if err != nil {
		return err
	}
	instance, err := redishost.Instance(fresh, record)
	if err != nil || instance.State != models.DatabaseReady || instance.Identity != record.ClusterID {
		return errors.New(DatabaseNotReady)
	}
	if _, err := backend.Statistics(fresh, record); err != nil {
		return err
	}
	if err := redisconnections.EnsurePrimary(fresh, record, operation.AccountID); err != nil {
		return err
	}
	if err := completeOperation(ctx, record, operation); err != nil {
		return err
	}
	if err := redistransfer.Cleanup(ctx, record); err != nil {
		logger.Warnf(LogPrefix, "Redis snapshot staging could not be removed")
	}
	proxy.Forget()
	if payload.Path != "" {
		_ = os.Remove(payload.Path)
	}
	return nil
}
