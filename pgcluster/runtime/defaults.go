package runtime

const (
	MajorVersion               = "18"
	ProcessName                = "postgres"
	ShutdownGroup              = 100
	secretNamespace            = "cluster"
	administratorSecretName    = "administrator"
	bootstrapFileName          = "bootstrap-password"
	configurationFileName      = "postgresql.conf"
	hostAuthenticationFileName = "pg_hba.conf"
	clusterLockFileName        = ".cluster.lock"
)
