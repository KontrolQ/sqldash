package explorer

import (
	"context"
	"net/http"
	"strings"

	"sqldash/sqld"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func UpdateCell(requestContext context.Context, databaseName string, table string, column string, key string, value string, clearIt bool) *fiber.Error {
	tables, tablesError := Tables(requestContext, databaseName)
	if tablesError != nil {
		return tablesError
	}

	if !known(tables, table) {
		return shortcuts.ServiceError(http.StatusNotFound, TableMissing)
	}

	for _, held := range tables {
		if held.Name == table && held.IsView {
			return shortcuts.ServiceError(http.StatusBadRequest, NotEditable)
		}
	}

	columns, columnsError := Columns(requestContext, databaseName, table)
	if columnsError != nil {
		return columnsError
	}

	if !columnNamed(columns, column) {
		return shortcuts.ServiceError(http.StatusBadRequest, ColumnMissing)
	}

	keyColumn := keyColumnOf(requestContext, databaseName, table, columns, false)
	if keyColumn == "" {
		return shortcuts.ServiceError(http.StatusBadRequest, NotEditable)
	}

	statement := "UPDATE " + quoteIdentifier(table) +
		" SET " + quoteIdentifier(column) + " = ?" +
		" WHERE " + quoteIdentifier(keyColumn) + " = ?"

	var wanted any = value
	if clearIt {
		wanted = nil
	}

	if _, runError := sqld.Query(requestContext, databaseName, statement, wanted, key); runError != nil {
		logger.Errorf(LogPrefix, WriteFailedLog, databaseName, runError)
		return shortcuts.ServiceError(http.StatusBadRequest, strings.TrimSpace(runError.Error()))
	}

	return nil
}

func DeleteRows(requestContext context.Context, databaseName string, table string, keys []string) *fiber.Error {
	if len(keys) == 0 {
		return shortcuts.ServiceError(http.StatusBadRequest, NothingChosen)
	}

	columns, columnsError := Columns(requestContext, databaseName, table)
	if columnsError != nil {
		return columnsError
	}

	keyColumn := keyColumnOf(requestContext, databaseName, table, columns, false)
	if keyColumn == "" {
		return shortcuts.ServiceError(http.StatusBadRequest, NotEditable)
	}

	places := make([]string, 0, len(keys))
	arguments := make([]any, 0, len(keys))

	for _, key := range keys {
		places = append(places, "?")
		arguments = append(arguments, key)
	}

	statement := "DELETE FROM " + quoteIdentifier(table) +
		" WHERE " + quoteIdentifier(keyColumn) + " IN (" + strings.Join(places, ", ") + ")"

	if _, runError := sqld.Query(requestContext, databaseName, statement, arguments...); runError != nil {
		logger.Errorf(LogPrefix, WriteFailedLog, databaseName, runError)
		return shortcuts.ServiceError(http.StatusBadRequest, strings.TrimSpace(runError.Error()))
	}

	return nil
}
