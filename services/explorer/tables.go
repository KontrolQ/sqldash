package explorer

import (
	"context"
	"fmt"
	"net/http"

	"sqldash/sqld"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func Tables(requestContext context.Context, databaseName string) ([]TableView, *fiber.Error) {
	held, queryError := sqld.Query(requestContext, databaseName, TableListSQL)
	if queryError != nil {
		logger.Errorf(LogPrefix, ReadFailedLog, databaseName, queryError)
		return nil, shortcuts.ServiceError(http.StatusBadGateway, TablesUnavailable)
	}

	views := make([]TableView, 0, len(held.Rows))

	for _, row := range held.Rows {
		views = append(views, TableView{
			Name:   fmt.Sprint(row[0]),
			IsView: fmt.Sprint(row[1]) == "view",
		})
	}

	return views, nil
}

func Columns(requestContext context.Context, databaseName string, table string) ([]ColumnView, *fiber.Error) {
	held, queryError := sqld.Query(requestContext, databaseName, fmt.Sprintf(ColumnListSQL, quoteIdentifier(table)))
	if queryError != nil {
		logger.Errorf(LogPrefix, ReadFailedLog, databaseName, queryError)
		return nil, shortcuts.ServiceError(http.StatusBadGateway, TablesUnavailable)
	}

	views := make([]ColumnView, 0, len(held.Rows))

	for _, row := range held.Rows {
		view := ColumnView{
			Name:       fmt.Sprint(row[1]),
			Kind:       fmt.Sprint(row[2]),
			NotNull:    fmt.Sprint(row[3]) == "1",
			PrimaryKey: fmt.Sprint(row[5]) != "0",
		}

		if row[4] != nil {
			view.Default = fmt.Sprint(row[4])
		}

		views = append(views, view)
	}

	attachForeignKeys(requestContext, databaseName, table, views)

	return views, nil
}

func attachForeignKeys(requestContext context.Context, databaseName string, table string, columns []ColumnView) {
	held, queryError := sqld.Query(requestContext, databaseName, fmt.Sprintf(ForeignKeyListSQL, quoteIdentifier(table)))
	if queryError != nil {
		return
	}

	for _, row := range held.Rows {
		target := fmt.Sprint(row[2])
		from := fmt.Sprint(row[3])

		to := ""
		if row[4] != nil {
			to = fmt.Sprint(row[4])
		}

		for index := range columns {
			if columns[index].Name == from {
				columns[index].References = target
				columns[index].OnColumn = to
			}
		}
	}
}
