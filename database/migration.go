package database

import (
	"context"
	"time"

	"sqldash/models"
	"sqldash/pictures"

	"gorm.io/gorm/clause"
)

func migrate(ctx context.Context) error {
	if migrationError := DB.WithContext(ctx).AutoMigrate(
		&models.Account{}, &models.AccountPicture{},
		&models.Site{}, &models.Role{}, &models.RolePermission{}, &models.AccountRole{},
		&models.DatabaseShare{}, &models.AccountLink{}, &models.OwnershipTransfer{}, &models.SMTPSettings{},
		&models.Session{}, &models.Database{}, &models.AccessRule{}, &models.Token{},
		&models.Fingerprint{}, &models.Statement{}, &models.Rollup{}, &models.Marker{},
		&models.Snippet{}, &models.Event{}, &models.PostgresConnection{}, &models.DatabaseOperation{},
		&models.SchemaMigration{}, &models.PostgresStatistics{}, &models.PostgresStatementStatistics{},
		&models.RedisInstance{}, &models.RedisConnection{}, &models.RedisStatistics{},
	); migrationError != nil {
		return migrationError
	}
	if markerError := DB.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&models.SchemaMigration{Name: SchemaReadyMarker, AppliedAt: time.Now().UTC()}).Error; markerError != nil {
		return markerError
	}
	var references []string
	for _, model := range []any{&models.Account{}, &models.AccountPicture{}} {
		var retained []string
		if readingError := DB.WithContext(ctx).Unscoped().Model(model).Where("profile_picture LIKE ?", pictures.ReferencePrefix+"%").Pluck("profile_picture", &retained).Error; readingError != nil {
			return readingError
		}
		references = append(references, retained...)
	}
	return pictures.Prune(references)
}
