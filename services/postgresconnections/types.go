package postgresconnections

import "sqldash/utils/collections"

type IndexContext struct {
	Title             string
	Name              string
	Engine            string
	EngineLabel       string
	Actions           map[string]bool
	Connections       []ConnectionView
	Scopes            []collections.Option
	Host              string
	Port              int
	Database          string
	DatabaseID        uint
	Issued            *IssuedPassword
	Username          string
	CredentialURL     string
	PasswordAvailable bool
	PasswordProblem   string
	PrimaryIdentifier uint
}

type ConnectionView struct {
	Identifier uint
	Label      string
	Username   string
	Scope      string
	State      string
	Expiry     string
	LastUsed   string
	URL        string
	CanRotate  bool
	CanRevoke  bool
}

type IssuedPassword struct {
	DatabaseID   uint
	ConnectionID uint
	Generation   uint64
	Label        string
	Username     string
	Password     string
	Host         string
	Port         int
	Database     string
	URL          string
}
