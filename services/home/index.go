package home

import (
	"net/http"

	"sqldash/database"
	"sqldash/models"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func GetIndexData() (*IndexContext, *fiber.Error) {
	records := make([]models.Database, 0)

	if findError := database.DB.Order(NameOrder).Find(&records).Error; findError != nil {
		logger.Errorf(LogPrefix, DatabaseListFailedLog, findError)
		return nil, shortcuts.ServiceError(http.StatusInternalServerError, DatabaseListFailed)
	}

	return &IndexContext{
		Title:     IndexTitle,
		Databases: toDatabaseViews(records),
	}, nil
}

func toDatabaseViews(records []models.Database) []DatabaseView {
	views := make([]DatabaseView, 0, len(records))

	for _, record := range records {
		views = append(views, DatabaseView{
			Name:      record.Name,
			Protected: record.Protected,
		})
	}

	return views
}
