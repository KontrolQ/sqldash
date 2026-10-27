package databases

import (
	"sqldash/services/postgresconnections/overviewhelper"
	"sqldash/services/redisconnections"
	tokenservice "sqldash/services/tokens"
	"sqldash/utils/collections"
)

type IndexContext struct {
	Title              string
	Databases          []DatabaseView
	Engine             string
	EngineFilters      []collections.Option
	CreateEngines      []collections.Option
	ImportEngines      []collections.Option
	CanCreateDatabases bool
	CanImportDatabases bool
	CanImportLibSQL    bool
	CanImportPostgres  bool
	CanImportRedis     bool
	ObjectHeading      string
	CanCloneRemote     bool
	RemoteTLSModes     []collections.Option
	RemoteCloneForm    *RemoteCloneForm
}

type RemoteCloneForm struct {
	Name     string
	Host     string
	Port     string
	Database string
	Username string
	SSLMode  string
	Error    string
}

type DatabaseView struct {
	DisplayName string
	Name        string
	Engine      string
	EngineLabel string
	State       string
	Address     string
	Protected   bool
	Blocked     bool
	Online      bool
	Ready       bool
	Storage     string
	Tables      string
	Objects     string
	RowsRead    string
	Queries     string
	Actions     map[string]bool
}

type ShowContext struct {
	Title                              string
	Name                               string
	Engine                             string
	EngineLabel                        string
	State                              string
	Ready                              bool
	Online                             bool
	Actions                            map[string]bool
	Address                            string
	URL                                string
	Protected                          bool
	BlockReads                         bool
	BlockWrites                        bool
	BlockReason                        string
	AllowAttach                        bool
	SettingsAvailable                  bool
	Storage                            string
	Tables                             string
	RowsRead                           string
	RowsWritten                        string
	QueryCount                         string
	Connections                        string
	Transactions                       string
	ConnectionLimit                    int
	StatementTimeoutMilliseconds       int
	IdleTransactionTimeoutMilliseconds int
	Host                               string
	Port                               int
	Username                           string
	PostgresCredentials                *overviewhelper.Credentials
	RedisCredentials                   *redisconnections.Issued
	MemoryUsed                         string
	MemoryLimit                        string
	MemoryAmount                       string
	MemoryUnit                         string
	MemoryUnits                        []collections.Option
	MemoryLimitBytes                   string
	RedisMemoryBudget                  string
	RedisMemoryOverhead                string
	EvictionPolicy                     string
	Persistence                        string
	Fsync                              string
	EvictionPolicies                   []collections.Option
	PersistenceOptions                 []collections.Option
	FsyncOptions                       []collections.Option
	PlaintextAllowed                   bool
	ManagedRedisCA                     bool
	PasswordProblem                    string
	CanCloneRemote                     bool
	RemoteTLSModes                     []collections.Option
	Tokens                             []tokenservice.TokenView
	Secret                             string
	Scopes                             []collections.Option
	Rules                              []RuleView
}

type RuleView struct {
	Identifier uint
	Network    string
	Note       string
}
