package explorer

import (
	"context"
	"net/http"
	"strings"

	sqld "sqldash/engines"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func InsertRow(requestContext context.Context, databaseName string, table string, values map[string]string, schemas ...string) *fiber.Error {
	record, permissionError := authorizedDatabase(requestContext, databaseName, "libsql.data.write")
	if permissionError != nil {
		return permissionError
	}
	if record.Kind() == "postgres" {
		fields := make(map[string]FieldValue, len(values))
		for name, value := range values {
			fields[name] = FieldValue{State: "value", Value: value}
		}
		return insertPostgres(requestContext, record, selectedSchema(schemas), table, fields)
	}
	tables, tablesError := Tables(requestContext, databaseName)
	if tablesError != nil {
		return tablesError
	}

	if !known(tables, table) {
		return shortcuts.ServiceError(http.StatusNotFound, TableMissing)
	}

	columns, columnsError := Columns(requestContext, databaseName, table)
	if columnsError != nil {
		return columnsError
	}

	names := make([]string, 0, len(values))
	marks := make([]string, 0, len(values))
	arguments := make([]any, 0, len(values))

	for _, column := range columns {
		held, given := values[column.Name]
		if !given || held == "" {
			continue
		}

		names = append(names, quoteIdentifier(column.Name))
		marks = append(marks, "?")
		arguments = append(arguments, held)
	}

	if len(names) == 0 {
		return shortcuts.ServiceError(http.StatusBadRequest, NothingToInsert)
	}

	statement := "INSERT INTO " + quoteIdentifier(table) +
		" (" + strings.Join(names, ", ") + ") VALUES (" + strings.Join(marks, ", ") + ")"

	if _, runError := sqld.Query(requestContext, databaseName, statement, arguments...); runError != nil {
		logger.Errorf(LogPrefix, WriteFailedLog, databaseName, runError)
		return shortcuts.ServiceError(http.StatusBadRequest, strings.TrimSpace(runError.Error()))
	}

	return nil
}
