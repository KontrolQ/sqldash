package secretstore

const (
	PostgresCredentialNamespace = "postgres-credential-pending"

	KeySize            = 32
	DirectoryMode      = 0o700
	FileMode           = 0o600
	keyFileName        = "key"
	lockFileName       = ".lock"
	secretExtension    = ".secret"
	maximumSecretBytes = 16 * 1024 * 1024
)
