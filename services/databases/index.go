package databases

import (
	"net/http"

	"sqldash/config"
	"sqldash/models"
	repository "sqldash/repositories/database"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func GetIndexData() (*IndexContext, *fiber.Error) {
	records, findError := repository.All()
	if findError != nil {
		logger.Errorf(LogPrefix, ListFailedLog, findError)
		return nil, shortcuts.ServiceError(http.StatusInternalServerError, ListUnavailable)
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
			Address:   record.Name + "." + config.Server.Domain,
			Protected: record.Protected,
		})
	}

	return views
}
