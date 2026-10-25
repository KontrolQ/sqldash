package databases

import (
	"context"
	"errors"
	"net/http"

	"sqldash/models"
	"sqldash/permissions"
	"sqldash/redishost"
	"sqldash/services/access"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func SaveRedisSettings(ctx context.Context, name string, settings redishost.Settings, protected bool) *fiber.Error {
	fresh, record, err := Authorize(ctx, name, permissions.RedisDatabaseSettings, true)
	if err != nil {
		return err
	}
	if record.Kind() != models.EngineRedis {
		return shortcuts.ServiceError(http.StatusBadRequest, EngineRefused)
	}
	checkAuthority := func(step context.Context) error {
		_, current, err := Authorize(step, name, permissions.RedisDatabaseSettings, true)
		if err != nil {
			return err
		}
		if current.ID != record.ID || current.ClusterID != record.ClusterID {
			return errors.New(AuthorityRefused)
		}
		return nil
	}
	guarded, stop, guardError := access.Guard(fresh, checkAuthority, nil)
	if guardError != nil {
		return shortcuts.ServiceError(http.StatusForbidden, AuthorityRefused)
	}
	defer stop()
	if err := redishost.UpdateSettings(guarded, record, settings, checkAuthority); err != nil {
		if checkAuthority(fresh) != nil {
			return shortcuts.ServiceError(http.StatusForbidden, AuthorityRefused)
		}
		if errors.Is(err, redishost.InvalidSettings) {
			return shortcuts.ServiceError(http.StatusBadRequest, SettingsRefused)
		}
		if errors.Is(err, redishost.CapacityExceeded) || errors.Is(err, redishost.PolicyUnconfirmed) {
			return shortcuts.ServiceError(http.StatusConflict, SettingsRefused)
		}
		return shortcuts.ServiceError(http.StatusBadGateway, SettingsRefused)
	}
	return saveProtection(fresh, record, protected)
}
