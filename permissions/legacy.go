package permissions

var LegacyPermissions = map[string]string{
	"databases.view":     DatabaseView,
	"databases.create":   DatabaseCreate,
	"databases.settings": DatabaseSettings,
	"databases.delete":   DatabaseDelete,
	"databases.import":   DatabaseImport,
	"databases.export":   DatabaseExport,
	"databases.fork":     DatabaseFork,
	"databases.share":    DatabaseShare,
	"data.read":          DataRead,
	"data.write":         DataWrite,
	"sql.execute":        SQLExecute,
	"tokens.manage":      TokenManage,
	"network.manage":     NetworkManage,
	"insights.view":      InsightsView,
	"audit.view":         AuditView,
}

var LegacyViewer = []string{"databases.view", "data.read", "insights.view"}
var LegacyEditor = []string{"databases.view", "databases.create", "databases.settings", "databases.import", "databases.export", "databases.fork", "data.read", "data.write", "sql.execute", "insights.view"}
