package explorer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"sqldash/sqld"

	"github.com/gofiber/fiber/v3"
)

func RunConsole(requestContext context.Context, databaseName string, statement string) (*ConsoleContext, *fiber.Error) {
	tables, tablesError := Tables(requestContext, databaseName)
	if tablesError != nil {
		return nil, tablesError
	}

	shown := &ConsoleContext{
		Title:      ConsoleTitle,
		Database:   databaseName,
		Tables:     tables,
		Statement:  statement,
		BrowsePath: DatabasePath + databaseName + BrowseSuffix,
		Schema:     schemaOf(requestContext, databaseName),
	}

	statement = strings.TrimSpace(statement)
	if statement == "" {
		return shown, nil
	}

	shown.Ran = true

	held, runError := sqld.Query(requestContext, databaseName, statement)
	if runError != nil {
		shown.Problem = strings.TrimSpace(runError.Error())
		return shown, nil
	}

	for _, column := range held.Columns {
		shown.Columns = append(shown.Columns, column.Name)
	}

	for _, row := range held.Rows {
		line := make([]string, 0, len(row))

		for _, cell := range row {
			line = append(line, readableCell(cell))
		}

		shown.Rows = append(shown.Rows, line)
	}

	shown.Affected = held.Affected
	shown.RowsRead = held.RowsRead
	shown.Duration = fmt.Sprintf(DurationFormat, held.DurationMs)

	return shown, nil
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

func schemaOf(requestContext context.Context, databaseName string) string {
	held, queryError := sqld.Query(requestContext, databaseName, SchemaSQL)
	if queryError != nil {
		return EmptySchema
	}

	order := make([]string, 0)
	columns := make(map[string][]string)

	for _, row := range held.Rows {
		if row[0] == nil || row[1] == nil {
			continue
		}

		table := fmt.Sprint(row[0])

		if _, seen := columns[table]; !seen {
			order = append(order, table)
		}

		columns[table] = append(columns[table], fmt.Sprint(row[1]))
	}

	tables := make([]SchemaTable, 0, len(order))

	for _, table := range order {
		tables = append(tables, SchemaTable{Name: table, Columns: columns[table]})
	}

	encoded, encodeError := json.Marshal(tables)
	if encodeError != nil {
		return EmptySchema
	}

	return string(encoded)
}
