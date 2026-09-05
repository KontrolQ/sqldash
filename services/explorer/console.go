package explorer

import (
	"context"
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
		Title:     ConsoleTitle,
		Database:  databaseName,
		Tables:    tables,
		Statement: statement,
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
