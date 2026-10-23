package postgresmanagement

import "time"

type ActivityView struct {
	PID            uint32
	Username       string
	Account        string
	Label          string
	Application    string
	ClientAddress  string
	StartedAt      string
	BackendStart   string
	State          string
	Wait           string
	Query          string
	QueryAge       string
	TransactionAge string
	Manageable     bool
}

type ActivityContext struct {
	Title         string
	Name          string
	Rows          []ActivityView
	CollectedAt   string
	Connections   int
	ActiveQueries int
	CanManage     bool
}

type SessionTarget struct {
	PID          uint32
	BackendStart time.Time
}

type ExtensionView struct {
	Label            string
	Name             string
	DefaultVersion   string
	InstalledVersion string
	Description      string
	Supported        bool
	Protected        bool
	Available        bool
	Enabled          bool
	CanChange        bool
	Source           string
	NativeUsage      string
	Reason           string
	Type             string
}

type ExtensionsContext struct {
	Title string
	Name  string
	Rows  []ExtensionView
}
