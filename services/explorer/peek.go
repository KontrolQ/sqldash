package explorer

import (
	"context"
	"fmt"
	"strings"

	"sqldash/sqld"
)

func attachPeeks(requestContext context.Context, databaseName string, columns []ColumnView, rows []RowView) {
	for index, column := range columns {
		if column.References == "" || column.OnColumn == "" {
			continue
		}

		wanted := distinctAt(rows, index)
		if len(wanted) == 0 {
			continue
		}

		found := lookUp(requestContext, databaseName, column.References, column.OnColumn, wanted)
		if found == nil {
			continue
		}

		for rowIndex := range rows {
			cell := &rows[rowIndex].Cells[index]

			if held, matched := found[cell.Text]; matched {
				cell.Peek = held
			}
		}
	}
}

func distinctAt(rows []RowView, index int) []any {
	seen := make(map[string]bool, len(rows))
	wanted := make([]any, 0, len(rows))

	for _, row := range rows {
		if index >= len(row.Cells) {
			continue
		}

		cell := row.Cells[index]

		if cell.IsNull || cell.Text == "" || seen[cell.Text] {
			continue
		}

		seen[cell.Text] = true
		wanted = append(wanted, cell.Text)

		if len(wanted) >= MaximumPeeks {
			break
		}
	}

	return wanted
}

func lookUp(requestContext context.Context, databaseName string, table string, column string, wanted []any) map[string]*PeekView {
	holders := strings.TrimSuffix(strings.Repeat("?, ", len(wanted)), ", ")

	statement := fmt.Sprintf(
		"SELECT * FROM %s WHERE %s IN (%s) LIMIT %d",
		quoteIdentifier(table), quoteIdentifier(column), holders, MaximumPeeks,
	)

	held, queryError := sqld.Query(requestContext, databaseName, statement, wanted...)
	if queryError != nil {
		return nil
	}

	names := make([]string, 0, len(held.Columns))
	at := -1

	for index, one := range held.Columns {
		names = append(names, one.Name)

		if one.Name == column {
			at = index
		}
	}

	if at < 0 {
		return nil
	}

	found := make(map[string]*PeekView, len(held.Rows))

	for _, row := range held.Rows {
		cells := make([]PeekCell, 0, len(row))

		for _, value := range row {
			cells = append(cells, peekCell(value))
		}

		found[fmt.Sprint(row[at])] = &PeekView{Table: table, Columns: names, Cells: cells}
	}

	return found
}

func peekCell(value any) PeekCell {
	if value == nil {
		return PeekCell{IsNull: true}
	}

	if held, isBytes := value.([]byte); isBytes {
		return PeekCell{Text: fmt.Sprintf(BytesFormat, len(held))}
	}

	text := fmt.Sprint(value)

	if len(text) > MaximumCellText {
		return PeekCell{Text: text[:MaximumCellText]}
	}

	return PeekCell{Text: text}
}
