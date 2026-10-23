package audit

const (
	LogPrefix = "Audit"

	Title    = "Audit log"
	PageSize = 25

	MomentLayout = "2 Jan 2006, 15:04"
	DayHours     = 24

	JustNow    = "just now"
	MinuteWord = "minute"
	HourWord   = "hour"
	DayWord    = "day"
	AgoSuffix  = " ago"
	Plural     = "s"

	EveryAction         = "Every action"
	EveryEngine         = "Every permitted engine"
	PlatformEngine      = "platform"
	DatabaseParameter   = "database"
	DatabaseIDParameter = "database_id"
	EngineParameter     = "engine"
	ActionParameter     = "action"
	EveryDatabase       = "every database"
	UnknownActor        = "unknown"
)

const (
	DatabaseCreated    = "database.created"
	DatabaseImported   = "database.imported"
	DatabaseForked     = "database.forked"
	DatabaseDeleted    = "database.deleted"
	DatabaseExported   = "database.exported"
	SettingsSaved      = "settings.saved"
	TokenMinted        = "token.minted"
	TokenRevoked       = "token.revoked"
	NetworkAllowed     = "network.allowed"
	NetworkRemoved     = "network.removed"
	RowsChanged        = "rows.changed"
	RowsDeleted        = "rows.deleted"
	RowInserted        = "row.inserted"
	StatementRun       = "statement.run"
	StatementCancelled = "statement.cancelled"
	SnippetSaved       = "snippet.saved"
	SnippetRemoved     = "snippet.removed"
	CredentialViewed   = "postgres.connection.viewed"
	CredentialRevealed = "postgres.connection.revealed"
	DatabaseCloned     = "database.cloned.remote"
	ObjectCreated      = "postgres.object.created"
	ObjectDropped      = "postgres.object.dropped"
	ObjectRenamed      = "postgres.object.renamed"
	ObjectChanged      = "postgres.object.changed"
	ObjectRefreshed    = "postgres.object.refreshed"
	AccountChanged     = "account.changed"
	PasswordChanged    = "password.changed"
	SignedIn           = "session.opened"
	SignedOut          = "session.closed"
)

const (
	GoodTone  = "good"
	BadTone   = "bad"
	PlainTone = "plain"
)
