package permissions

const (
	RedisDatabaseView      = "redis.databases.view"
	RedisDatabaseCreate    = "redis.databases.create"
	RedisDatabaseSettings  = "redis.databases.settings"
	RedisDatabaseDelete    = "redis.databases.delete"
	RedisDatabaseImport    = "redis.databases.import"
	RedisDatabaseExport    = "redis.databases.export"
	RedisDatabaseFork      = "redis.databases.fork"
	RedisDatabaseShare     = "redis.databases.share"
	RedisDataRead          = "redis.data.read"
	RedisDataWrite         = "redis.data.write"
	RedisConsoleExecute    = "redis.console.execute"
	RedisCredentialsManage = "redis.credentials.manage"
	RedisNetworkManage     = "redis.network.manage"
	RedisInsightsView      = "redis.insights.view"
	RedisAuditView         = "redis.audit.view"
	RedisActivityView      = "redis.activity.view"
	RedisActivityManage    = "redis.activity.manage"
	RedisSlowLogView       = "redis.slowlog.view"
	RedisSlowLogManage     = "redis.slowlog.manage"
	RedisMonitorView       = "redis.monitor.view"
	RedisSubscribe         = "redis.pubsub.subscribe"
	RedisPublish           = "redis.pubsub.publish"
)

func init() {
	Catalogue = append(Catalogue,
		Capability{RedisDatabaseView, "View databases", "Redis · Databases", "See accessible Redis databases and their configuration.", "redis"},
		Capability{RedisDatabaseCreate, "Create databases", "Redis · Databases", "Create managed Redis databases.", "redis"},
		Capability{RedisDatabaseSettings, "Edit database settings", "Redis · Databases", "Change memory, eviction, persistence and delete protection.", "redis"},
		Capability{RedisDatabaseDelete, "Delete databases", "Redis · Databases", "Permanently remove accessible, unprotected Redis databases.", "redis"},
		Capability{RedisDatabaseImport, "Import databases", "Redis · Databases", "Create Redis databases from supported snapshots.", "redis"},
		Capability{RedisDatabaseExport, "Export databases", "Redis · Databases", "Download consistent Redis snapshots and supported key exports.", "redis"},
		Capability{RedisDatabaseFork, "Copy databases", "Redis · Databases", "Create a separate Redis database from a current snapshot.", "redis"},
		Capability{RedisDatabaseShare, "Manage database sharing", "Redis · Databases", "Share Redis databases with users or remove their access.", "redis"},
		Capability{RedisDataRead, "Read data", "Redis · Data & connections", "Browse keys, values, collections and expiry information.", "redis"},
		Capability{RedisDataWrite, "Write data", "Redis · Data & connections", "Create, edit, rename and delete keys. Requires Read data.", "redis"},
		Capability{RedisConsoleExecute, "Execute commands", "Redis · Data & connections", "Run Redis commands. Writes also require Write data.", "redis"},
		Capability{RedisCredentialsManage, "Manage database credentials", "Redis · Data & connections", "Issue, rotate and revoke scoped native Redis credentials.", "redis"},
		Capability{RedisNetworkManage, "Manage network rules", "Redis · Data & connections", "Change source-address ranges allowed to connect.", "redis"},
		Capability{RedisInsightsView, "View insights", "Redis · Monitoring", "See Redis health, activity and command statistics.", "redis"},
		Capability{RedisAuditView, "View audit log", "Redis · Monitoring", "Read recorded Redis management activity.", "redis"},
		Capability{RedisActivityView, "View connections", "Redis · Monitoring", "Inspect database-scoped Redis clients.", "redis"},
		Capability{RedisActivityManage, "Manage connections", "Redis · Monitoring", "Disconnect Redis clients. Requires View connections.", "redis"},
		Capability{RedisSlowLogView, "View slow log", "Redis · Monitoring", "Inspect slow commands and their potentially sensitive arguments.", "redis"},
		Capability{RedisSlowLogManage, "Manage slow log", "Redis · Monitoring", "Reset the slow log. Requires View slow log.", "redis"},
		Capability{RedisMonitorView, "Monitor commands", "Redis · Monitoring", "Explicitly stream database commands and application values.", "redis"},
		Capability{RedisSubscribe, "Subscribe to channels", "Redis · Pub/Sub", "Receive live messages from this database's channels.", "redis"},
		Capability{RedisPublish, "Publish messages", "Redis · Pub/Sub", "Publish messages to this database's channels.", "redis"},
	)
}
