package database

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"runtime"

	"sqldash/config"

	"gorm.io/gorm"
)

const pictureBackupSuffix = ".before-pictures"

var sqliteBackupHeader = []byte("SQLite format 3\x00")

func preparePictureBackup(ctx context.Context, store *gorm.DB) error {
	path := config.StatePath() + pictureBackupSuffix
	if inspectionError := verifyPictureBackup(path); inspectionError == nil {
		return syncPictureBackupDirectory()
	} else if !errors.Is(inspectionError, os.ErrNotExist) {
		return inspectionError
	}
	file, creationError := os.CreateTemp(config.Data.Directory, ".pictures-backup-")
	if creationError != nil {
		return creationError
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if closingError := file.Close(); closingError != nil {
		return closingError
	}
	if backupError := store.WithContext(ctx).Exec("VACUUM INTO ?", temporary).Error; backupError != nil {
		return backupError
	}
	file, openingError := os.OpenFile(temporary, os.O_RDWR, 0600)
	if openingError != nil {
		return openingError
	}
	syncError := file.Sync()
	closingError := file.Close()
	if syncError != nil || closingError != nil {
		return errors.Join(syncError, closingError)
	}
	if linkingError := os.Link(temporary, path); linkingError != nil && !errors.Is(linkingError, os.ErrExist) {
		return linkingError
	}
	if verificationError := verifyPictureBackup(path); verificationError != nil {
		return verificationError
	}
	return syncPictureBackupDirectory()
}

func syncPictureBackupDirectory() error {
	if runtime.GOOS == "windows" {
		return nil
	}
	directory, openingError := os.Open(config.Data.Directory)
	if openingError != nil {
		return openingError
	}
	syncError := directory.Sync()
	closingError := directory.Close()
	return errors.Join(syncError, closingError)
}

func verifyPictureBackup(path string) error {
	information, inspectionError := os.Lstat(path)
	if inspectionError != nil {
		return inspectionError
	}
	if !information.Mode().IsRegular() || information.Size() < 512 {
		return errors.New(PictureBackupInvalid)
	}
	file, openingError := os.Open(path)
	if openingError != nil {
		return openingError
	}
	defer file.Close()
	header := make([]byte, len(sqliteBackupHeader))
	if _, readingError := io.ReadFull(file, header); readingError != nil || !bytes.Equal(header, sqliteBackupHeader) {
		return errors.New(PictureBackupInvalid)
	}
	return nil
}
