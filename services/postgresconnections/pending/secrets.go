package pending

import (
	"context"
	"encoding/json"
	"errors"

	"sqldash/config"
	"sqldash/database"
	"sqldash/models"
	"sqldash/secretstore"
	"sqldash/services/postgresconnections/mutation"
	primarycredentials "sqldash/services/postgresconnections/primary"
)

func Clear(ctx context.Context, databaseID, connectionID uint, references ...string) error {
	if databaseID == 0 {
		return errors.New("invalid database identity")
	}
	return mutation.With(ctx, databaseID, func(fenced context.Context) error {
		return clearReferences(fenced, databaseID, connectionID, references...)
	})
}

func clearReferences(ctx context.Context, databaseID, connectionID uint, references ...string) error {
	temporary := make(map[string]bool)
	primary := make(map[string]bool)
	for _, reference := range references {
		if reference != "" {
			temporary[reference] = true
		}
	}
	query := database.DB.WithContext(ctx).Where("database_id = ?", databaseID)
	if connectionID != 0 {
		query = query.Where("id = ?", connectionID)
	}
	var connections []models.PostgresConnection
	if err := query.Find(&connections).Error; err != nil {
		return err
	}
	for _, connection := range connections {
		if connection.PrimaryConnection {
			if connection.PendingSecret != "" {
				primary[connection.PendingSecret] = true
			}
			if connection.PasswordReference != "" {
				primary[connection.PasswordReference] = true
			}
		} else if connection.PendingSecret != "" {
			temporary[connection.PendingSecret] = true
		}
	}
	operationsQuery := database.DB.WithContext(ctx).Where("database_id = ? AND kind IN ?", databaseID, []string{"credential-create", "credential-rotate", "credential-revoke"})
	if connectionID != 0 {
		operationsQuery = operationsQuery.Where("connection_id = ?", connectionID)
	}
	var operations []models.DatabaseOperation
	if err := operationsQuery.Find(&operations).Error; err != nil {
		return err
	}
	for _, operation := range operations {
		var payload struct {
			SecretReference           string
			PreviousSecret            string
			PreviousPasswordReference string
			PrimaryConnection         bool
		}
		if err := json.Unmarshal([]byte(operation.Payload), &payload); err != nil {
			return err
		}
		if payload.PrimaryConnection {
			if payload.SecretReference != "" {
				primary[payload.SecretReference] = true
			}
			if payload.PreviousSecret != "" {
				primary[payload.PreviousSecret] = true
			}
			if payload.PreviousPasswordReference != "" {
				primary[payload.PreviousPasswordReference] = true
			}
		} else {
			if payload.SecretReference != "" {
				temporary[payload.SecretReference] = true
			}
			if payload.PreviousSecret != "" {
				temporary[payload.PreviousSecret] = true
			}
		}
	}
	if connectionID == 0 {
		primary[primarycredentials.Reference(databaseID, 1)] = true
	}
	if len(temporary) == 0 && len(primary) == 0 {
		return nil
	}
	store, err := secretstore.Open(config.PostgresSecretsPath())
	if err != nil {
		return err
	}
	for name := range temporary {
		if err := store.Delete(secretstore.PostgresCredentialNamespace, name); err != nil {
			return err
		}
	}
	for name := range primary {
		if err := store.Delete(primarycredentials.SecretNamespace, name); err != nil {
			return err
		}
	}
	return nil
}
