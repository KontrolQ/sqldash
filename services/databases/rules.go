package databases

import (
	"context"
	"net"
	"net/http"
	"strings"

	"sqldash/database"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func AllowNetwork(ctx context.Context, databaseName, network, note string) *fiber.Error {
	fresh, record, err := Authorize(ctx, databaseName, permissions.NetworkManage, true)
	if err != nil {
		return err
	}
	_, parsed, parseError := net.ParseCIDR(strings.TrimSpace(network))
	if parseError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, NetworkUnusable)
	}
	rule := &models.AccessRule{DatabaseName: record.Name, Network: parsed.String(), Note: strings.TrimSpace(note)}
	if err := database.DB.WithContext(fresh).Transaction(func(transaction *gorm.DB) error {
		if err := transaction.Create(rule).Error; err != nil {
			return err
		}
		return transaction.Model(record).Update("policy_version", gorm.Expr("policy_version + 1")).Error
	}); err != nil {
		return shortcuts.ServiceError(http.StatusInternalServerError, NetworkRefused)
	}
	return nil
}

func RemoveNetwork(ctx context.Context, databaseName string, identifier uint) *fiber.Error {
	fresh, record, err := Authorize(ctx, databaseName, permissions.NetworkManage, true)
	if err != nil {
		return err
	}
	err = nil
	var deleted bool
	removeError := database.DB.WithContext(fresh).Transaction(func(transaction *gorm.DB) error {
		result := transaction.Where("id = ? AND database_name = ?", identifier, record.Name).Delete(&models.AccessRule{})
		if result.Error != nil {
			return result.Error
		}
		deleted = result.RowsAffected == 1
		if !deleted {
			return nil
		}
		return transaction.Model(record).Update("policy_version", gorm.Expr("policy_version + 1")).Error
	})
	if removeError != nil {
		return shortcuts.ServiceError(http.StatusInternalServerError, NetworkRefused)
	}
	if !deleted {
		return shortcuts.ServiceError(http.StatusNotFound, NetworkMissing)
	}
	return nil
}
