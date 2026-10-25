package access

import (
	"sqldash/models"
	"sqldash/permissions"
)

var DatabaseActions = map[string]string{
	"CanViewDatabases":        permissions.DatabaseView,
	"CanCreateDatabases":      permissions.DatabaseCreate,
	"CanReadData":             permissions.DataRead,
	"CanWriteData":            permissions.DataWrite,
	"CanRunSQL":               permissions.SQLExecute,
	"CanManageTokens":         permissions.TokenManage,
	"CanManageNetwork":        permissions.NetworkManage,
	"CanEditDatabaseSettings": permissions.DatabaseSettings,
	"CanDeleteDatabases":      permissions.DatabaseDelete,
	"CanImportDatabases":      permissions.DatabaseImport,
	"CanExportDatabases":      permissions.DatabaseExport,
	"CanForkDatabases":        permissions.DatabaseFork,
	"CanViewInsights":         permissions.InsightsView,
	"CanViewAudit":            permissions.AuditView,
}

func Actions(held *AccountAccess, record *models.Database) map[string]bool {
	flags := make(map[string]bool, len(DatabaseActions)+6)
	for name, permission := range DatabaseActions {
		flags[name] = held.CanDatabase(record, permission)
	}
	flags["CanShareDatabase"] = held.CanShare(record)
	flags["CanManageConnections"] = record.Kind() == models.EnginePostgres && held.Can(permissions.PostgresCredentialsManage)
	flags["CanManageObjects"] = record.Kind() == models.EnginePostgres && held.Can(permissions.PostgresObjectsManage)
	flags["CanViewActivity"] = record.Kind() == models.EnginePostgres && held.Can(permissions.PostgresActivityView)
	flags["CanManageActivity"] = record.Kind() == models.EnginePostgres && held.Can(permissions.PostgresActivityManage)
	flags["CanManageExtensions"] = record.Kind() == models.EnginePostgres && held.Can(permissions.PostgresExtensionsManage)
	if record.Kind() == models.EngineRedis {
		flags["CanManageConnections"] = held.Can(permissions.RedisCredentialsManage)
		flags["CanViewActivity"] = held.Can(permissions.RedisActivityView)
		flags["CanManageActivity"] = held.Can(permissions.RedisActivityManage)
	}
	flags["CanViewSlowLog"] = record.Kind() == models.EngineRedis && held.Can(permissions.RedisSlowLogView)
	flags["CanResetSlowLog"] = record.Kind() == models.EngineRedis && held.Can(permissions.RedisSlowLogManage)
	flags["CanMonitorRedis"] = record.Kind() == models.EngineRedis && held.Can(permissions.RedisMonitorView)
	flags["CanSubscribeRedis"] = record.Kind() == models.EngineRedis && held.Can(permissions.RedisSubscribe)
	flags["CanPublishRedis"] = record.Kind() == models.EngineRedis && held.Can(permissions.RedisPublish)
	return flags
}
