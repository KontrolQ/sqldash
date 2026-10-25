package router

import (
	controller "sqldash/controllers/redis"
	page "sqldash/pages/redis"
	"sqldash/permissions"
	"sqldash/utils/urls"
)

func init() {
	urls.SetNamespace("")
	urls.Path(urls.Get, "/databases/:name/slow-log", page.SlowLog, "databases.redis.slowlog")
	urls.Path(urls.Get, "/databases/:name/monitor", page.Monitor, "databases.redis.monitor")
	urls.Path(urls.Get, "/databases/:name/pubsub", page.PubSub, "databases.redis.pubsub")
	urls.Path(urls.Post, "/databases/:name/redis/keys/scan", controller.Scan, "databases.redis.keys.scan")
	urls.Path(urls.Post, "/databases/:name/redis/keys/read", controller.Read, "databases.redis.keys.read")
	urls.Path(urls.Post, "/databases/:name/redis/keys/mutate", controller.Mutate, "databases.redis.keys.mutate")
	urls.Path(urls.Post, "/databases/:name/redis/keys/batch", controller.MutateBatch, "databases.redis.keys.batch")
	urls.Path(urls.Post, "/databases/:name/redis/keys/rename", controller.Rename, "databases.redis.keys.rename")
	urls.Path(urls.Post, "/databases/:name/redis/keys/delete", controller.Delete, "databases.redis.keys.delete")
	urls.Path(urls.Post, "/databases/:name/redis/console", controller.Console, "databases.redis.console.execute")
	urls.Path(urls.Get, "/databases/:name/redis/insights", controller.Insights, "databases.redis.insights")
	urls.Path(urls.Get, "/databases/:name/redis/commands", controller.Commands, "databases.redis.commands")
	urls.Path(urls.Get, "/databases/:name/redis/slow-log", controller.SlowLog, "databases.redis.slowlog.read")
	urls.Path(urls.Post, "/databases/:name/redis/slow-log/reset", controller.ResetSlowLog, "databases.redis.slowlog.reset")
	urls.Path(urls.Post, "/databases/:name/redis/monitor", controller.Monitor, "databases.redis.monitor.stream")
	urls.Path(urls.Post, "/databases/:name/redis/pubsub/subscribe", controller.Subscribe, "databases.redis.pubsub.subscribe")
	urls.Path(urls.Post, "/databases/:name/redis/pubsub/publish", controller.Publish, "databases.redis.pubsub.publish")
	urls.Path(urls.Get, "/databases/:name/redis/clients", controller.Clients, "databases.redis.clients")
	urls.Path(urls.Post, "/databases/:name/redis/clients/terminate", controller.TerminateClient, "databases.redis.clients.terminate")
}

func RedisRouteCapability(name string) string {
	switch name {
	case "databases.redis.settings":
		return permissions.RedisDatabaseSettings
	case "databases.redis.overview.rotate.primary", "databases.redis.connections.create", "databases.redis.connections.rotate", "databases.redis.connections.revoke", "databases.redis.connections.rotate.primary", "databases.redis.connections.reveal":
		return permissions.RedisCredentialsManage
	case "databases.redis.keys.scan", "databases.redis.keys.read":
		return permissions.RedisDataRead
	case "databases.redis.keys.mutate", "databases.redis.keys.batch", "databases.redis.keys.rename", "databases.redis.keys.delete":
		return permissions.RedisDataWrite
	case "databases.redis.console.execute":
		return permissions.RedisConsoleExecute
	case "databases.redis.insights", "databases.redis.commands":
		return permissions.RedisInsightsView
	case "databases.redis.slowlog", "databases.redis.slowlog.read":
		return permissions.RedisSlowLogView
	case "databases.redis.slowlog.reset":
		return permissions.RedisSlowLogManage
	case "databases.redis.monitor", "databases.redis.monitor.stream":
		return permissions.RedisMonitorView
	case "databases.redis.pubsub", "databases.redis.pubsub.subscribe":
		return permissions.RedisSubscribe
	case "databases.redis.pubsub.publish":
		return permissions.RedisPublish
	case "databases.redis.clients":
		return permissions.RedisActivityView
	case "databases.redis.clients.terminate":
		return permissions.RedisActivityManage
	}
	return ""
}
