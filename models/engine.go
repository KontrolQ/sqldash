package models

const (
	EngineLibSQL     = "libsql"
	EnginePostgres   = "postgres"
	EngineRedis      = "redis"
	DatabasePending  = "pending"
	DatabaseApplying = "applying"
	DatabaseReady    = "ready"
	DatabaseFailed   = "failed"
	DatabaseDeleting = "deleting"
)

func (self *Database) Kind() string {
	if self.Engine == "" {
		return EngineLibSQL
	}
	return self.Engine
}

func IsKnownEngine(engine string) bool {
	return engine == EngineLibSQL || engine == EnginePostgres || engine == EngineRedis
}

func (self *Database) Available() bool {
	return (self.State == "" || self.State == DatabaseReady) && IsKnownEngine(self.Kind())
}
