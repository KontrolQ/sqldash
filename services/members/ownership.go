package members

import (
	"gorm.io/gorm"
	"sqldash/database"
	"sqldash/models"
	"sqldash/services/access"
	"sqldash/services/mailing"
	"sqldash/utils/passwords"
	"time"
)

func TransferOwnership(actor *access.AccountAccess, recipientID uint, password string) error {
	if actor == nil {
		return Refused
	}
	recipient, recipientError := access.Load(recipientID)
	if recipientError != nil || !recipient.Account.Active {
		return Invalid
	}
	createError := database.DB.Transaction(func(transaction *gorm.DB) error {
		current, findError := access.LoadWith(transaction, actor.Account.ID)
		if findError != nil || !current.Owner || !current.Account.Active || recipientID == current.Account.ID || !passwords.Matches(password, current.Account.PasswordHash) {
			return Refused
		}
		var target models.Account
		if findError := transaction.First(&target, recipientID).Error; findError != nil || !target.Active {
			return Invalid
		}
		if cancelError := transaction.Model(&models.OwnershipTransfer{}).Where("status = ?", Pending).Update("status", "cancelled").Error; cancelError != nil {
			return cancelError
		}
		return transaction.Create(&models.OwnershipTransfer{OwnerAccountID: actor.Account.ID, RecipientAccountID: recipientID, Status: Pending, ExpiresAt: time.Now().Add(TransferLifetime)}).Error
	})
	if createError == nil {
		mailing.Send(recipient.Account.Email, OwnershipSubject, "The site owner has requested an ownership transfer. Sign in to sqldash and open your account settings to accept or decline it.")
	}
	return createError
}

func RespondOwnership(actor *access.AccountAccess, identifier uint, action string, password string) error {
	if actor == nil {
		return Refused
	}
	return database.DB.Transaction(func(transaction *gorm.DB) error {
		current, findError := access.LoadWith(transaction, actor.Account.ID)
		if findError != nil || !current.Account.Active {
			return Refused
		}
		actor = current
		var request models.OwnershipTransfer
		if findError := transaction.Where("id = ? AND status = ? AND expires_at > ?", identifier, Pending, time.Now()).First(&request).Error; findError != nil {
			return Invalid
		}
		var site models.Site
		if findError := transaction.First(&site, 1).Error; findError != nil {
			return findError
		}
		if site.OwnerAccountID != request.OwnerAccountID {
			return Refused
		}
		switch action {
		case "cancel":
			if site.OwnerAccountID != actor.Account.ID {
				return Refused
			}
		case "decline", "accept":
			if request.RecipientAccountID != actor.Account.ID || !actor.Account.Active {
				return Refused
			}
			if action == "accept" {
				var recipient models.Account
				if findError := transaction.First(&recipient, actor.Account.ID).Error; findError != nil {
					return findError
				}
				if !recipient.Active || !passwords.Matches(password, recipient.PasswordHash) {
					return PasswordWrong
				}
				updated := transaction.Model(&models.Site{}).Where("id = 1 AND owner_account_id = ?", request.OwnerAccountID).Update("owner_account_id", actor.Account.ID)
				if updated.Error != nil {
					return updated.Error
				}
				if updated.RowsAffected != 1 {
					return Refused
				}
			}
		default:
			return Invalid
		}
		status := map[string]string{"accept": "accepted", "decline": "declined", "cancel": "cancelled"}[action]
		updated := transaction.Model(&request).Where("status = ?", Pending).Update("status", status)
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return Invalid
		}
		return nil
	})
}
