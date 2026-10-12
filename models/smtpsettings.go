package models

type SMTPSettings struct {
	ID             uint `gorm:"primaryKey"`
	Host           string
	Port           int
	Security       string
	Username       string
	PasswordCipher string `gorm:"type:text"`
	FromEmail      string
	FromName       string
}
