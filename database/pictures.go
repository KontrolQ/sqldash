package database

import (
	"context"

	"sqldash/models"
	"sqldash/pictures"
	"sqldash/utils/logger"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const pictureMigrationBatchSize = 32

type legacyPicture struct {
	ID                 uint
	ProfilePicture     string
	ProfilePictureType string
}

func migratePicturesWith(ctx context.Context, store *gorm.DB) error {
	store = store.WithContext(ctx).Session(&gorm.Session{Logger: store.Logger.LogMode(gormlogger.Silent)})
	backupPrepared := false
	migrated := int64(0)
	for _, model := range []any{&models.Account{}, &models.AccountPicture{}} {
		for {
			var records []legacyPicture
			query := store.WithContext(ctx).Unscoped().Model(model).Select("id", "profile_picture", "profile_picture_type").Where("profile_picture <> '' AND profile_picture NOT LIKE ?", pictures.ReferencePrefix+"%").Order("id asc").Limit(pictureMigrationBatchSize)
			if readingError := query.Find(&records).Error; readingError != nil {
				return readingError
			}
			if len(records) == 0 {
				break
			}
			if !backupPrepared {
				if backupError := preparePictureBackup(ctx, store); backupError != nil {
					return backupError
				}
				backupPrepared = true
			}
			for _, record := range records {
				reference, storageError := pictures.Store(record.ProfilePicture, record.ProfilePictureType)
				if storageError != nil {
					return storageError
				}
				updated := store.WithContext(ctx).Unscoped().Model(model).Where("id = ? AND profile_picture = ? AND profile_picture_type = ?", record.ID, record.ProfilePicture, record.ProfilePictureType).UpdateColumn("profile_picture", reference)
				if updated.Error != nil {
					return updated.Error
				}
				migrated += updated.RowsAffected
			}
		}
	}
	var references []string
	for _, model := range []any{&models.Account{}, &models.AccountPicture{}} {
		var retained []string
		if readingError := store.WithContext(ctx).Unscoped().Model(model).Where("profile_picture LIKE ?", pictures.ReferencePrefix+"%").Pluck("profile_picture", &retained).Error; readingError != nil {
			return readingError
		}
		references = append(references, retained...)
	}
	if cleanupError := pictures.Prune(references); cleanupError != nil {
		return cleanupError
	}
	if migrated > 0 {
		logger.Infof(LogPrefix, PictureMigrationCompletedLog, migrated)
	}
	return nil
}
