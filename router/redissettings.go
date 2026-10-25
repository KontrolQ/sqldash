package router

import (
	controllers "sqldash/controllers/databases"
	"sqldash/utils/urls"
)

func init() {
	urls.SetNamespace("")
	urls.Path(urls.Post, "/databases/:name/settings/redis", controllers.SaveRedisSettings, "databases.redis.settings")
	urls.Path(urls.Post, "/databases/:name/settings/redis/password/rotate", controllers.RotateRedisPrimary, "databases.redis.overview.rotate.primary")
}
