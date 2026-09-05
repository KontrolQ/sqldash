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

func Structure(requestContext context.Context, databaseName string, table string) (*StructureContext, *fiber.Error) {
	tables, tablesError := Tables(requestContext, databaseName)
	if tablesError != nil {
		return nil, tablesError
	}

	shown := &StructureContext{
		Title:         databaseName,
		Database:      databaseName,
		Tables:        tables,
		BrowsePath:    DatabasePath + databaseName + BrowseSuffix,
		StructurePath: DatabasePath + databaseName + StructureSuffix,
	}

	if table == "" {
		return shown, nil
	}

	if !known(tables, table) {
		return nil, shortcuts.ServiceError(http.StatusNotFound, TableMissing)
	}

	shown.Table = table
	shown.Title = table

	for _, one := range tables {
		if one.Name == table {
			shown.IsView = one.IsView
		}
	}

	columns, columnsError := Columns(requestContext, databaseName, table)
	if columnsError != nil {
		return nil, columnsError
	}

	shown.Columns = columns
	shown.Indexes = indexesOf(requestContext, databaseName, table)
	shown.Definition = definitionOf(requestContext, databaseName, table)

	return shown, nil
}

func indexesOf(requestContext context.Context, databaseName string, table string) []IndexView {
	held, queryError := sqld.Query(requestContext, databaseName, fmt.Sprintf(IndexListSQL, quoteIdentifier(table)))
	if queryError != nil {
		logger.Warnf(LogPrefix, ReadFailedLog, databaseName, queryError)
		return nil
	}

	views := make([]IndexView, 0, len(held.Rows))

	for _, row := range held.Rows {
		name := fmt.Sprint(row[1])

		view := IndexView{
			Name:   name,
			Unique: fmt.Sprint(row[2]) == "1",
			Origin: fmt.Sprint(row[3]),
		}

		if parts, partsError := sqld.Query(requestContext, databaseName, fmt.Sprintf(IndexInfoSQL, quoteIdentifier(name))); partsError == nil {
			for _, part := range parts.Rows {
				if part[2] != nil {
					view.Columns = append(view.Columns, fmt.Sprint(part[2]))
				}
			}
		}

		views = append(views, view)
	}

	return views
}

func definitionOf(requestContext context.Context, databaseName string, table string) string {
	held, queryError := sqld.Query(requestContext, databaseName, DefinitionSQL, table)
	if queryError != nil || len(held.Rows) == 0 || held.Rows[0][0] == nil {
		return ""
	}

	return fmt.Sprint(held.Rows[0][0])
}
