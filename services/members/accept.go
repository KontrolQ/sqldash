package members

import (
	"strings"
	"time"

	"sqldash/database"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/services/access"
	"sqldash/utils/passwords"

	"gorm.io/gorm"
)

func consumeLink(transaction *gorm.DB, kind string, secret string) (*models.AccountLink, error) {
	link, findError := findLinkWith(transaction, kind, secret)
	if findError != nil {
		return nil, findError
	}
	now := time.Now()
	changed := transaction.Model(&models.AccountLink{}).Where("id = ? AND used_at IS NULL AND revoked_at IS NULL AND expires_at > ?", link.ID, now).Update("used_at", now)
	if changed.Error != nil {
		return nil, changed.Error
	}
	if changed.RowsAffected != 1 {
		return nil, LinkInvalid
	}
	return link, nil
}

func AcceptInvitation(secret string, username string, displayName string, password string) (*models.Account, error) {
	username = strings.TrimSpace(username)
	displayName = strings.TrimSpace(displayName)
	if username == "" || len(username) > 80 || len(displayName) > 120 {
		return nil, Invalid
	}
	if passwordError := passwords.Acceptable(password); passwordError != nil {
		return nil, passwordError
	}
	if _, findError := FindLink(Invitation, secret); findError != nil {
		return nil, findError
	}
	hash, hashError := passwords.Hash(password)
	if hashError != nil {
		return nil, hashError
	}
	var account models.Account
	acceptError := database.DB.Transaction(func(transaction *gorm.DB) error {
		link, linkError := consumeLink(transaction, Invitation, secret)
		if linkError != nil {
			return linkError
		}
		issuer, issuerError := access.LoadWith(transaction, link.IssuerID)
		if issuerError != nil || !issuer.Can(permissions.UsersManage) {
			return Refused
		}
		account = models.Account{Username: username, DisplayName: displayName, Email: link.Email, PasswordHash: hash, Active: true}
		if createError := transaction.Create(&account).Error; createError != nil {
			return createError
		}
		return nil
	})
	if acceptError != nil {
		return nil, acceptError
	}
	return &account, nil
}

func RecoverAccount(secret string, password string) error {
	if passwordError := passwords.Acceptable(password); passwordError != nil {
		return passwordError
	}
	if _, findError := FindLink(Recovery, secret); findError != nil {
		return findError
	}
	hash, hashError := passwords.Hash(password)
	if hashError != nil {
		return hashError
	}
	return database.DB.Transaction(func(transaction *gorm.DB) error {
		link, linkError := consumeLink(transaction, Recovery, secret)
		if linkError != nil {
			return linkError
		}
		issuer, issuerError := access.LoadWith(transaction, link.IssuerID)
		target, targetError := access.LoadWith(transaction, link.AccountID)
		if issuerError != nil || targetError != nil || !issuer.CanManage(target) || !target.Account.Active {
			return Refused
		}
		changed := transaction.Model(&models.Account{}).Where("id = ? AND active = ?", link.AccountID, true).Updates(map[string]any{"password_hash": hash, "session_version": gorm.Expr("session_version + 1")})
		if changed.Error != nil {
			return changed.Error
		}
		if changed.RowsAffected != 1 {
			return Refused
		}
		return nil
	})
}
