package sqld

import "time"

const (
	LogPrefix = "Sqld"

	AdminTimeout         = 30 * time.Second
	MaximumAdminResponse = 4 << 20

	NamespacePrefix = "/v1/namespaces"
	NamespaceHeader = "x-namespace"

	CreateAction = "create"
	ForkAction   = "fork"
	StatsAction  = "stats"
	ConfigAction = "config"

	TimestampParameter = "timestamp"
	DumpField          = "dump_url"
	DumpScheme         = "file://"

	ContentTypeHeader = "content-type"
	ApplicationJSON   = "application/json"
)
