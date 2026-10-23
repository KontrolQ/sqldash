package permissions

const (
	LogPrefix                 = "Permissions"
	SuperAdminKind            = "super-admin"
	SuperAdminPosition        = 1000000
	DatabaseView              = "libsql.databases.view"
	DatabaseCreate            = "libsql.databases.create"
	DatabaseSettings          = "libsql.databases.settings"
	DatabaseDelete            = "libsql.databases.delete"
	DatabaseImport            = "libsql.databases.import"
	DatabaseExport            = "libsql.databases.export"
	DatabaseFork              = "libsql.databases.fork"
	DatabaseShare             = "libsql.databases.share"
	DataRead                  = "libsql.data.read"
	DataWrite                 = "libsql.data.write"
	SQLExecute                = "libsql.sql.execute"
	TokenManage               = "libsql.tokens.manage"
	NetworkManage             = "libsql.network.manage"
	InsightsView              = "libsql.insights.view"
	AuditView                 = "libsql.audit.view"
	PostgresDatabaseView      = "postgres.databases.view"
	PostgresDatabaseCreate    = "postgres.databases.create"
	PostgresDatabaseSettings  = "postgres.databases.settings"
	PostgresDatabaseDelete    = "postgres.databases.delete"
	PostgresDatabaseImport    = "postgres.databases.import"
	PostgresDatabaseExport    = "postgres.databases.export"
	PostgresDatabaseFork      = "postgres.databases.fork"
	PostgresDatabaseShare     = "postgres.databases.share"
	PostgresDataRead          = "postgres.data.read"
	PostgresDataWrite         = "postgres.data.write"
	PostgresSQLExecute        = "postgres.sql.execute"
	PostgresCredentialsManage = "postgres.credentials.manage"
	PostgresObjectsManage     = "postgres.objects.manage"
	PostgresNetworkManage     = "postgres.network.manage"
	PostgresInsightsView      = "postgres.insights.view"
	PostgresAuditView         = "postgres.audit.view"
	PostgresActivityView      = "postgres.activity.view"
	PostgresActivityManage    = "postgres.activity.manage"
	PostgresExtensionsManage  = "postgres.extensions.manage"
	PlatformAuditView         = "audit.view"
	UsersManage               = "users.manage"
	RolesManage               = "roles.manage"
	SMTPManage                = "settings.smtp"
	Administrator             = "administrator"
)

type Capability struct {
	Key         string
	Label       string
	Group       string
	Description string
	Engine      string
}

var Catalogue = []Capability{
	{DatabaseView, "View databases", "libSQL · Databases", "See accessible databases and their overview pages.", "libsql"},
	{DatabaseCreate, "Create databases", "libSQL · Databases", "Create new libSQL databases.", "libsql"},
	{DatabaseSettings, "Edit database settings", "libSQL · Databases", "Change read/write blocks and delete protection.", "libsql"},
	{DatabaseDelete, "Delete databases", "libSQL · Databases", "Permanently remove accessible, unprotected databases.", "libsql"},
	{DatabaseImport, "Import databases", "libSQL · Databases", "Create databases from SQLite files or SQL dumps.", "libsql"},
	{DatabaseExport, "Export databases", "libSQL · Databases", "Download databases as SQLite files or SQL dumps.", "libsql"},
	{DatabaseFork, "Fork databases", "libSQL · Databases", "Copy accessible databases.", "libsql"},
	{DatabaseShare, "Manage database sharing", "libSQL · Databases", "Share databases with users or remove their access.", "libsql"},
	{DataRead, "Read data", "libSQL · Data & connections", "Browse rows, tables and database structure.", "libsql"},
	{DataWrite, "Write data", "libSQL · Data & connections", "Insert, edit and delete rows.", "libsql"},
	{SQLExecute, "Execute SQL", "libSQL · Data & connections", "Run SQL in the console. Requires Write data.", "libsql"},
	{TokenManage, "Manage application tokens", "libSQL · Data & connections", "Create and revoke client tokens. Requires Write data.", "libsql"},
	{NetworkManage, "Manage network rules", "libSQL · Data & connections", "Change the source-address ranges allowed to connect.", "libsql"},
	{InsightsView, "View insights", "libSQL · Monitoring", "See query performance for accessible libSQL databases.", "libsql"},
	{AuditView, "View audit log", "libSQL · Monitoring", "Read recorded activity for accessible libSQL databases.", "libsql"},
	{PostgresDatabaseView, "View databases", "PostgreSQL · Databases", "See accessible databases and their overview pages.", "postgres"},
	{PostgresDatabaseCreate, "Create databases", "PostgreSQL · Databases", "Create new PostgreSQL databases.", "postgres"},
	{PostgresDatabaseSettings, "Edit database settings", "PostgreSQL · Databases", "Change database protection and connection settings.", "postgres"},
	{PostgresDatabaseDelete, "Delete databases", "PostgreSQL · Databases", "Permanently remove accessible, unprotected databases.", "postgres"},
	{PostgresDatabaseImport, "Import databases", "PostgreSQL · Databases", "Create databases from supported PostgreSQL dumps.", "postgres"},
	{PostgresDatabaseExport, "Export databases", "PostgreSQL · Databases", "Download PostgreSQL SQL dumps and backup archives.", "postgres"},
	{PostgresDatabaseFork, "Clone databases", "PostgreSQL · Databases", "Create a copy from the current database snapshot.", "postgres"},
	{PostgresDatabaseShare, "Manage database sharing", "PostgreSQL · Databases", "Share databases with users or remove their access.", "postgres"},
	{PostgresDataRead, "Read data", "PostgreSQL · Data & connections", "Browse schemas, tables and rows.", "postgres"},
	{PostgresDataWrite, "Write data", "PostgreSQL · Data & connections", "Insert, edit and delete rows. Requires Read data.", "postgres"},
	{PostgresSQLExecute, "Execute SQL", "PostgreSQL · Data & connections", "Run SQL in the console. Requires Read data; changing data also requires Write data.", "postgres"},
	{PostgresCredentialsManage, "Manage database credentials", "PostgreSQL · Data & connections", "View the database connection string and rotate its generated password. Requires Read data and Write data.", "postgres"},
	{PostgresObjectsManage, "Manage database structure", "PostgreSQL · Data & connections", "Create and change database objects. Requires Write data.", "postgres"},
	{PostgresNetworkManage, "Manage network rules", "PostgreSQL · Data & connections", "Change the source-address ranges allowed to connect.", "postgres"},
	{PostgresInsightsView, "View insights", "PostgreSQL · Monitoring", "See query statistics for accessible PostgreSQL databases.", "postgres"},
	{PostgresAuditView, "View audit log", "PostgreSQL · Monitoring", "Read recorded activity for accessible PostgreSQL databases.", "postgres"},
	{PostgresActivityView, "View connections", "PostgreSQL · Monitoring", "Inspect active connections and queries.", "postgres"},
	{PostgresActivityManage, "Manage connections", "PostgreSQL · Monitoring", "Cancel queries and terminate database connections.", "postgres"},
	{PostgresExtensionsManage, "Manage extensions", "PostgreSQL · Databases", "Install or remove supported database extensions.", "postgres"},
	{PlatformAuditView, "View administration audit log", "Administration", "Read recorded account and administration activity.", ""},
	{UsersManage, "Manage users", "Administration", "Invite users and manage accounts below this role's authority.", ""},
	{RolesManage, "Manage roles", "Administration", "Create and edit lower roles using permissions this user already holds.", ""},
	{SMTPManage, "Manage SMTP settings", "Administration", "View and edit mail-server credentials, and send test emails.", ""},
	{Administrator, "Administrator", "Administration", "Full access to databases, users and instance settings.", ""},
}

var Viewer = []string{DatabaseView, DataRead, InsightsView}
var Editor = []string{DatabaseView, DatabaseCreate, DatabaseSettings, DatabaseImport, DatabaseExport, DatabaseFork, DataRead, DataWrite, SQLExecute, InsightsView}

func Known(key string) bool {
	for _, capability := range Catalogue {
		if capability.Key == key {
			return true
		}
	}
	return false
}
