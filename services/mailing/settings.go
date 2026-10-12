package mailing

import (
	"errors"
	"net/mail"
	"strings"

	"sqldash/database"
	"sqldash/models"

	"gorm.io/gorm"
)

func Settings() (*models.SMTPSettings, error) {
	var held models.SMTPSettings
	findError := database.DB.First(&held, 1).Error
	if errors.Is(findError, gorm.ErrRecordNotFound) {
		return &models.SMTPSettings{ID: 1, Port: 587, Security: "starttls"}, nil
	}
	return &held, findError
}

func Save(held models.SMTPSettings, password string) error {
	held.ID = 1
	held.Host = strings.TrimSpace(held.Host)
	held.FromEmail = strings.TrimSpace(held.FromEmail)
	if held.Host == "" || held.Port < 1 || held.Port > 65535 || (held.Security != "starttls" && held.Security != "tls") || strings.ContainsAny(held.Host+held.FromName+held.Username, "\r\n") {
		return SettingsInvalid
	}
	if address, parseError := mail.ParseAddress(held.FromEmail); parseError != nil || address.Address != held.FromEmail {
		return SettingsInvalid
	}
	previous, readError := Settings()
	if readError != nil {
		return readError
	}
	held.PasswordCipher = previous.PasswordCipher
	if password != "" {
		encrypted, encryptError := encrypt(password)
		if encryptError != nil {
			return encryptError
		}
		held.PasswordCipher = encrypted
	}
	return database.DB.Save(&held).Error
}

func Password(held *models.SMTPSettings) (string, error) {
	if held == nil {
		return "", nil
	}
	return decrypt(held.PasswordCipher)
}

func Clear() error { return database.DB.Where("id = ?", 1).Delete(&models.SMTPSettings{}).Error }
func Configured(held *models.SMTPSettings) bool {
	return held != nil && held.Host != "" && held.FromEmail != "" && held.Port > 0 && held.Port <= 65535 && (held.Security == "starttls" || held.Security == "tls")
}
