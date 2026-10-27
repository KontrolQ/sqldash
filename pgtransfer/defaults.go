package pgtransfer

import "time"

const (
	LogPrefix = "PostgreSQL transfer"

	FormatSQL    = "sql"
	FormatCustom = "custom"

	CleanupTimeout         = time.Minute
	ClientWaitDelay        = 2 * time.Second
	MaximumDiagnosticBytes = 16 * 1024
	MaximumVersionBytes    = 4096
	WrapperInspectionBytes = 8192
	CopyBufferBytes        = 64 * 1024
	RestrictionKeyBytes    = 32
	TransferUserID         = 65534
	TransferGroupID        = 65534
	PrivateDirectoryMode   = 0o700
	PrivateFileMode        = 0o600

	DumpClient              = "pg_dump"
	RestoreClient           = "pg_restore"
	SQLClient               = "psql"
	ArchiveMagic            = "PGDMP"
	MaximumArchiveListBytes = 16 * 1024 * 1024
	PlainHeader             = "--\n-- PostgreSQL database dump\n--\n\n"
	PlainFooter             = "--\n-- PostgreSQL database dump complete\n--\n\n"

	RemoteSourceLabel           = "remote PostgreSQL source"
	SSLVerifyFull               = "verify-full"
	SSLVerifyCA                 = "verify-ca"
	SSLRequire                  = "require"
	SSLDisable                  = "disable"
	MinimumRemoteServerVersion  = 90200
	MaximumRemoteServerVersion  = 190000
	MaximumCertificateBytes     = 1024 * 1024
	MaximumRemotePasswordBytes  = 4096
	MaximumSourceSchemas        = 4096
	MaximumSourceExtensions     = 256
	MaximumMetadataMessageBytes = 1024 * 1024
	RemoteConnectionTimeout     = 10 * time.Second
	SystemCertificatePath       = "/etc/ssl/certs/ca-certificates.crt"
)
