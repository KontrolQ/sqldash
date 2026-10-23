package primary

import (
	"context"
	"errors"
	"time"

	"sqldash/database"
	"sqldash/models"
	"sqldash/services/postgresconnections/mutation"

	"gorm.io/gorm"
)

func Mutate(ctx context.Context, record *models.Database, accountID uint, mutate func(context.Context) error) error {
	if record == nil || record.ID == 0 || record.Kind() != models.EnginePostgres || mutate == nil {
		return Unavailable
	}
	bounded, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	return mutation.With(bounded, record.ID, func(fenced context.Context) error {
		until := time.Now().Add(3 * time.Minute)
		fence := &models.DatabaseOperation{DatabaseID: record.ID, AccountID: accountID, Engine: models.EnginePostgres, Kind: "credential-fence", State: models.DatabaseApplying, LeaseUntil: &until}
		if err := database.DB.WithContext(fenced).Transaction(func(transaction *gorm.DB) error {
			var current models.Database
			if err := transaction.First(&current, record.ID).Error; err != nil {
				return err
			}
			if current.Kind() != models.EnginePostgres || current.State == models.DatabaseDeleting {
				return Unavailable
			}
			if err := transaction.Model(&models.DatabaseOperation{}).Where("database_id = ? AND kind = ? AND state = ?", record.ID, "credential-fence", models.DatabaseApplying).Updates(map[string]any{"state": models.DatabaseFailed, "lease_until": nil}).Error; err != nil {
				return err
			}
			return transaction.Create(fence).Error
		}); err != nil {
			return err
		}
		err := mutate(fenced)
		cleanup, cleanupCancel := context.WithTimeout(context.WithoutCancel(fenced), 10*time.Second)
		defer cleanupCancel()
		state := models.DatabaseReady
		if err != nil {
			state = models.DatabaseFailed
		}
		released := database.DB.WithContext(cleanup).Model(fence).Where("state = ? AND lease_until = ?", models.DatabaseApplying, until).Updates(map[string]any{"state": state, "lease_until": nil})
		if released.Error != nil {
			return errors.Join(err, released.Error)
		}
		if released.RowsAffected != 1 {
			return errors.Join(err, Unavailable)
		}
		return err
	})
}
