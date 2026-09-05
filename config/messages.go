package config

const (
	ParseFailedLog  = "Failed to read configuration from the environment: %v"
	VerifyFailedLog = "Failed to verify configuration: %v"
	LoadedLog       = "Configuration loaded for %s"
)

const (
	DomainMissing            = "A domain must be set."
	CertificateTokenMissing  = "Automatic certificates need a Cloudflare API token, because a wildcard can only be issued over DNS-01."
	CertificateEmailMissing  = "Automatic certificates need an email address to register with the certificate authority."
	DataDirectoryUnreachable = "The data directory could not be created."
)
