package router

import (
	connections "sqldash/controllers/redisconnections"
	"sqldash/utils/urls"
)

func init() {
	urls.SetNamespace("")
	urls.Path(urls.Post, "/databases/:name/connections/redis", connections.Create, "databases.redis.connections.create")
	urls.Path(urls.Post, "/databases/:name/connections/redis/primary/rotate", connections.RotatePrimary, "databases.redis.connections.rotate.primary")
	urls.Path(urls.Post, "/databases/:name/connections/redis/:identifier/rotate", connections.Rotate, "databases.redis.connections.rotate")
	urls.Path(urls.Post, "/databases/:name/connections/redis/:identifier/revoke", connections.Revoke, "databases.redis.connections.revoke")
	urls.Path(urls.Post, "/databases/:name/connections/redis/:identifier/reveal", connections.Reveal, "databases.redis.connections.reveal")
}
