package main

const (
	LogPrefix = "Sqldash"

	ModeProxy           = "proxy"
	ModeWeb             = "web"
	ModePostgresGateway = "postgres-proxy"
	ModeRedisHost       = "redis-host"
	ModeRedisGateway    = "redis-proxy"

	SqldProcessName  = "libsql server"
	ProxyProcessName = "proxy"
	WebProcessName   = "dashboard"

	TemplateRoot      = "./templates"
	TemplateExtension = ".django"
)
