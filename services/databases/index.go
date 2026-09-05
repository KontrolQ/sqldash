package databases

import (
	"context"
	"net/http"

	"sqldash/config"
	"sqldash/models"
	repository "sqldash/repositories/database"
	explorer "sqldash/services/explorer"
	"sqldash/sqld"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func GetIndexData(requestContext context.Context) (*IndexContext, *fiber.Error) {
	records, findError := repository.All()
	if findError != nil {
		logger.Errorf(LogPrefix, ListFailedLog, findError)
		return nil, shortcuts.ServiceError(http.StatusInternalServerError, ListUnavailable)
	}

	return &IndexContext{
		Title:     IndexTitle,
		Databases: toDatabaseViews(requestContext, records),
	}, nil
}

func toDatabaseViews(requestContext context.Context, records []models.Database) []DatabaseView {
	views := make([]DatabaseView, 0, len(records))

	for _, record := range records {
		view := DatabaseView{
			Name:      record.Name,
			Address:   record.Name + "." + config.Server.Domain,
			Protected: record.Protected,
			Storage:   UnknownSize,
			RowsRead:  NoNumber,
			Queries:   NoNumber,
		}

		if held, statisticsError := sqld.Statistics(requestContext, record.Name); statisticsError == nil {
			view.Storage = readableSize(held.StorageBytesUsed)
			view.RowsRead = readableCount(held.RowsRead)
			view.Queries = readableCount(held.QueryCount)
		}

		if settings, settingsError := sqld.Configuration(requestContext, record.Name); settingsError == nil {
			view.Blocked = settings.BlockReads || settings.BlockWrites
		}

		if tables, tablesError := explorer.Tables(requestContext, record.Name); tablesError == nil {
			view.Tables = len(tables)
		}

		views = append(views, view)
	}

	return views
}
