package models

const (
	EngineLibSQL     = "libsql"
	EnginePostgres   = "postgres"
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

func (self *Database) Available() bool {
	return (self.State == "" || self.State == DatabaseReady) && (self.Kind() == EngineLibSQL || self.Kind() == EnginePostgres)
}
