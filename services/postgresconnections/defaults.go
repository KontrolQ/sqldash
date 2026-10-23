package postgresconnections

import "sqldash/secretstore"

const (
	PendingNamespace = secretstore.PostgresCredentialNamespace
	IssuedPurpose    = "postgres-password"
	CreatedAction    = "postgres.connection.created"
	RotatedAction    = "postgres.connection.rotated"
	RevokedAction    = "postgres.connection.revoked"
	IndexTitle       = "Connections"
	ExpiryLayout     = "2006-01-02T15:04"
)
