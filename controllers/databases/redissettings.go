package databases

import (
	"fmt"
	"sqldash/utils/bytesize"

	"sqldash/redishost"
	"sqldash/services/audit"
	service "sqldash/services/databases"
	"sqldash/sessions"
	"sqldash/utils/meta"

	"github.com/gofiber/fiber/v3"
)

type RedisSettingsRequest struct {
	MemoryLimit    string `form:"memoryLimit"`
	MemoryUnit     string `form:"memoryUnit"`
	EvictionPolicy string `form:"evictionPolicy"`
	Persistence    string `form:"persistence"`
	Fsync          string `form:"fsync"`
	Protected      string `form:"protected"`
}

func SaveRedisSettings(context fiber.Ctx) error {
	name := context.Params(NameParameter)
	asked, err := meta.Body[RedisSettingsRequest](context)
	if err != nil {
		sessions.Complain(context, FormUnreadable)
		return backTo(context, name)
	}
	memoryInput := asked.MemoryLimit
	if asked.MemoryUnit != "" {
		memoryInput += " " + asked.MemoryUnit
	}
	memoryLimit, err := bytesize.Parse(memoryInput)
	if err != nil {
		sessions.Complain(context, bytesize.Invalid.Error())
		return backTo(context, name)
	}
	if asked.Protected != "" && asked.Protected != CheckedValue {
		sessions.Complain(context, FormUnreadable)
		return backTo(context, name)
	}
	settings := redishost.Settings{MemoryLimit: memoryLimit, EvictionPolicy: asked.EvictionPolicy, Persistence: asked.Persistence, Fsync: asked.Fsync}
	if err := service.SaveRedisSettings(context.Context(), name, settings, asked.Protected == CheckedValue); err != nil {
		sessions.Complain(context, err.Message)
	} else {
		sessions.Report(context, SettingsSaved)
		audit.Note(context, name, audit.SettingsSaved, fmt.Sprintf("Memory limit: %d bytes; eviction: %s; persistence: %s; fsync: %s; delete protection: %t", memoryLimit, settings.EvictionPolicy, settings.Persistence, settings.Fsync, asked.Protected == CheckedValue))
	}
	return backTo(context, name)
}
