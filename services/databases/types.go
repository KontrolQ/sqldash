package databases

import (
	"sqldash/services/postgresconnections/overviewhelper"
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
