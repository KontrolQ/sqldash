package explorer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"sqldash/config"
	"sqldash/engines"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

type ConsoleResult struct {
	Number            int
	Columns           []string
	Rows              [][]string
	Records           [][]PeekCell
	Affected          int64
	Duration          string
	RowsRead          int64
	RowsReadAvailable bool
}

func RunConsole(requestContext context.Context, databaseName string, statement string, options ...string) (*ConsoleContext, *fiber.Error) {
	record, actor, permissionError := authorizedDatabaseAccess(requestContext, databaseName, permissions.SQLExecute)
	if permissionError != nil {
		return nil, permissionError
	}
	schema := selectedSchema(options)
	if record.Kind() == models.EnginePostgres && !visiblePostgresSchema(schema) {
		return nil, consoleRequestError("Choose a user schema.")
	}
	tables, tablesError := Tables(requestContext, databaseName, schema)
	if tablesError != nil {
		return nil, tablesError
	}
	shown := &ConsoleContext{Title: ConsoleTitle, Database: databaseName, Engine: record.Kind(),
		Address: databaseName + "." + config.Server.Domain, Tables: tables, Statement: statement,
		BrowsePath: DatabasePath + url.PathEscape(databaseName) + BrowseSuffix, Schema: schemaOf(requestContext, databaseName),
		Snippets: Snippets(databaseName), CurrentSchema: schema, RowsReadAvailable: record.Kind() != models.EnginePostgres,
		CancelPath: DatabasePath + url.PathEscape(databaseName) + "/console/cancel"}
	if record.Kind() == models.EnginePostgres {
		shown.Address = config.PostgresHost()
		relations, relationsError := postgresRelations(requestContext, record, "")
		if relationsError != nil {
			return nil, relationsError
		}
		shown.Schemas, permissionError = postgresSchemas(requestContext, record, relations)
		if permissionError != nil {
			return nil, permissionError
		}
	}
	if strings.TrimSpace(statement) == "" {
		return shown, nil
	}
	shown.Ran = true
	mode := engines.SQLMode(actor.CanDatabase(record, permissions.DataWrite), actor.CanDatabase(record, permissions.PostgresObjectsManage))
	if record.Kind() == models.EngineLibSQL && mode == engines.ReadOnly {
		shown.Failure = "Native read-only SQL execution is not configured for this libSQL backend."
		return shown, nil
	}
	if record.Kind() == models.EnginePostgres {
		runID := ""
		if len(options) > 1 {
			runID = options[1]
		}
		var release func()
		requestContext, release, permissionError = registerConsole(requestContext, record, runID)
		if permissionError != nil {
			return nil, permissionError
		}
		defer release()
		shown.RunID = runID
	}
	results, runError := engines.Run(databaseContext(requestContext, record, mode), databaseName, engines.Statement{SQL: statement})
	if runError != nil {
		shown.Failure = strings.TrimSpace(runError.Error())
		return shown, nil
	}
	for index, result := range results {
		duration := fmt.Sprintf(DurationFormat, result.DurationMs)
		if record.Kind() == models.EnginePostgres && result.DurationMs == 0 {
			duration = "duration unavailable"
		}
		view := ConsoleResult{Number: index + 1, Affected: result.Affected, Duration: duration, RowsRead: result.RowsRead, RowsReadAvailable: shown.RowsReadAvailable}
		for _, column := range result.Columns {
			view.Columns = append(view.Columns, column.Name)
		}
		for _, row := range result.Rows {
			line := make([]string, 0, len(row))
			cells := make([]PeekCell, 0, len(row))
			for _, cell := range row {
				cells = append(cells, PeekCell{Text: postgresReadable(cell), IsNull: cell == nil})
				if record.Kind() == models.EnginePostgres {
					line = append(line, postgresReadable(cell))
				} else {
					line = append(line, readableCell(cell))
				}
			}
			view.Rows = append(view.Rows, line)
			view.Records = append(view.Records, cells)
		}
		shown.Results = append(shown.Results, view)
	}
	if len(shown.Results) > 0 {
		first := shown.Results[0]
		shown.Columns = first.Columns
		shown.Rows = first.Rows
		shown.Affected = first.Affected
		shown.RowsRead = first.RowsRead
		shown.Duration = first.Duration
	}
	return shown, nil
}

func ConsoleAccess(requestContext context.Context, name string) *fiber.Error {
	_, permissionError := authorizedDatabase(requestContext, name, permissions.SQLExecute)
	return permissionError
}

func readableCell(value any) string {
	if value == nil {
		return NullText
	}
	if held, isBytes := value.([]byte); isBytes {
		return fmt.Sprintf(BytesFormat, len(held))
	}
	text := fmt.Sprint(value)
	if len(text) > MaximumCellText {
		return text[:MaximumCellText] + Ellipsis
	}
	return text
}

func postgresReadable(value any) string {
	if value == nil {
		return NullText
	}
	if bytes, isBytes := value.([]byte); isBytes {
		return fmt.Sprintf("\\x%x", bytes)
	}
	return fmt.Sprint(value)
}

func schemaOf(requestContext context.Context, databaseName string) string {
	record, recordError := engines.Resolve(requestContext, databaseName)
	if recordError != nil {
		return EmptySchema
	}
	tables := make([]SchemaTable, 0)
	if record.Kind() == models.EnginePostgres {
		var tablesError *fiber.Error
		tables, tablesError = postgresCompletionTables(requestContext, record)
		if tablesError != nil {
			return EmptySchema
		}
	} else {
		result, queryError := engines.Query(requestContext, databaseName, SchemaSQL)
		if queryError != nil {
			return EmptySchema
		}
		positions := make(map[string]int)
		for _, row := range result.Rows {
			if row[0] == nil || row[1] == nil {
				continue
			}
			name := fmt.Sprint(row[0])
			position, seen := positions[name]
			if !seen {
				position = len(tables)
				positions[name] = position
				tables = append(tables, SchemaTable{Name: name})
			}
			tables[position].Columns = append(tables[position].Columns, fmt.Sprint(row[1]))
		}
	}
	encoded, encodeError := json.Marshal(tables)
	if encodeError != nil {
		return EmptySchema
	}
	return string(encoded)
}

func consoleRequestError(message string) *fiber.Error {
	return shortcuts.ServiceError(http.StatusBadRequest, message)
}
