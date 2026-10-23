package control

import (
	"context"

	"sqldash/models"
)

func ProvisionDefaults(ctx context.Context, record *models.Database) error {
	if record == nil || record.State != models.DatabaseApplying {
		return InvalidDatabase
	}
	if record.BackendOID == 0 || record.ClusterID == "" {
		return IdentityMismatch
	}
	return changeExtension(ctx, record, "pg_stat_statements", "", "", true)
}
