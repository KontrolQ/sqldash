package members

import (
	"sqldash/database"
	"sqldash/models"
	"sqldash/services/access"

	"gorm.io/gorm"
)

func SaveUser(actor *access.AccountAccess, accountID uint, roles []uint, databases []uint, all bool) error {
	return database.DB.Transaction(func(transaction *gorm.DB) error {
		current, findError := managedAccount(transaction, actor, accountID)
		if findError != nil {
			return findError
		}
		if grantError := validateGrantsWith(transaction, current, roles, databases, all); grantError != nil {
			return grantError
		}
		if !current.All() {
			visible := access.Scope(transaction.Model(&models.Database{}).Select("id"), current)
			var retained []models.DatabaseShare
			if findError := transaction.Where("account_id = ? AND database_id NOT IN (?)", accountID, visible).Find(&retained).Error; findError != nil {
				return findError
			}
			for _, share := range retained {
				databases = append(databases, share.DatabaseID)
			}
		}
		if assignError := Assign(transaction, accountID, roles, databases, current.Account.ID); assignError != nil {
			return assignError
		}
		return transaction.Model(&models.Account{}).Where("id = ?", accountID).Update("all_databases", all).Error
	})
}

func Activate(actor *access.AccountAccess, accountID uint, active bool) error {
	return database.DB.Transaction(func(transaction *gorm.DB) error {
		if _, findError := managedAccount(transaction, actor, accountID); findError != nil {
			return findError
		}
		if updateError := transaction.Model(&models.Account{}).Where("id = ?", accountID).Updates(map[string]any{"active": active, "session_version": gorm.Expr("session_version + 1")}).Error; updateError != nil {
			return updateError
		}
		if !active {
			if revokeError := transaction.Model(&models.AccountLink{}).Where("account_id = ? AND used_at IS NULL AND revoked_at IS NULL", accountID).Update("revoked_at", gorm.Expr("CURRENT_TIMESTAMP")).Error; revokeError != nil {
				return revokeError
			}
			return transaction.Model(&models.OwnershipTransfer{}).Where("recipient_account_id = ? AND status = ?", accountID, Pending).Update("status", "cancelled").Error
		}
		return nil
	})
}

func RevokeSessions(actor *access.AccountAccess, accountID uint) error {
	return database.DB.Transaction(func(transaction *gorm.DB) error {
		if _, findError := managedAccount(transaction, actor, accountID); findError != nil {
			return findError
		}
		return transaction.Model(&models.Account{}).Where("id = ?", accountID).Update("session_version", gorm.Expr("session_version + 1")).Error
	})
}
