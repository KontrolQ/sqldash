package permissions

const (
	LogPrefix          = "Permissions"
	SuperAdminKind     = "super-admin"
	SuperAdminPosition = 1000000
	DatabaseView       = "databases.view"
	DatabaseCreate     = "databases.create"
	DatabaseSettings   = "databases.settings"
	DatabaseDelete     = "databases.delete"
	DatabaseImport     = "databases.import"
	DatabaseExport     = "databases.export"
	DatabaseFork       = "databases.fork"
	DatabaseShare      = "databases.share"
	DataRead           = "data.read"
	DataWrite          = "data.write"
	SQLExecute         = "sql.execute"
	TokenManage        = "tokens.manage"
	NetworkManage      = "network.manage"
	InsightsView       = "insights.view"
	AuditView          = "audit.view"
	UsersManage        = "users.manage"
	RolesManage        = "roles.manage"
	SMTPManage         = "settings.smtp"
	Administrator      = "administrator"
)

type Capability struct {
	Key         string
	Label       string
	Group       string
	Description string
}

var Catalogue = []Capability{
	{DatabaseView, "View databases", "Databases", "See accessible databases and their overview pages."},
	{DatabaseCreate, "Create databases", "Databases", "Create new databases."},
	{DatabaseSettings, "Edit database settings", "Databases", "Change read/write blocks and delete protection."},
	{DatabaseDelete, "Delete databases", "Databases", "Permanently remove an accessible, unprotected database."},
	{DatabaseImport, "Import databases", "Databases", "Create a database from a SQLite file or SQL dump."},
	{DatabaseExport, "Export databases", "Databases", "Download accessible databases as SQLite files or SQL dumps."},
	{DatabaseFork, "Fork databases", "Databases", "Copy databases."},
	{DatabaseShare, "Manage database sharing", "Databases", "Share databases with users or remove their access."},
	{DataRead, "Read data", "Data & connections", "Browse rows, tables and database structure."},
	{DataWrite, "Write data", "Data & connections", "Insert, edit and delete rows."},
	{SQLExecute, "Execute SQL", "Data & connections", "Run SQL in the console. Requires Write data."},
	{TokenManage, "Manage application tokens", "Data & connections", "Create and revoke client tokens. Requires Write data."},
	{NetworkManage, "Manage network rules", "Data & connections", "Change the source-address ranges allowed to connect."},
	{InsightsView, "View insights", "Monitoring", "See query performance for accessible databases."},
	{AuditView, "View audit log", "Monitoring", "Read recorded activity within this user's database access."},
	{UsersManage, "Manage users", "Administration", "Invite users and manage accounts below this role's authority."},
	{RolesManage, "Manage roles", "Administration", "Create and edit lower roles using permissions this user already holds."},
	{SMTPManage, "Manage SMTP settings", "Administration", "View and edit mail-server credentials, and send test emails."},
	{Administrator, "Administrator", "Administration", "Full access to databases, users and instance settings."},
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
