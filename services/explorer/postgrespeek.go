package explorer

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"sqldash/engines"
	"sqldash/models"
)

func attachPostgresPeeks(requestContext context.Context, record *models.Database, constraints []engines.Constraint, columns []ColumnView, rows []RowView) {
	for _, constraint := range constraints {
		if !visiblePostgresSchema(constraint.ReferenceSchema) || constraint.ReferenceTable == "" || len(constraint.Columns) == 0 || len(constraint.Columns) != len(constraint.ReferenceColumns) {
			continue
		}
		targets, columnsError := postgresColumns(requestContext, record, constraint.ReferenceSchema, constraint.ReferenceTable)
		if columnsError != nil {
			continue
		}
		sourceIndexes := make([]int, 0, len(constraint.Columns))
		targetKeys := make([]ColumnView, 0, len(constraint.Columns))
		for index, source := range constraint.Columns {
			for columnIndex, column := range columns {
				if column.Name != source {
					continue
				}
				target, found := columnByName(targets, constraint.ReferenceColumns[index])
				if found {
					sourceIndexes = append(sourceIndexes, columnIndex)
					targetKeys = append(targetKeys, target)
				}
			}
		}
		if len(sourceIndexes) != len(constraint.Columns) {
			continue
		}
		wantedRows := make([]string, 0)
		arguments := make([]any, 0)
		for rowIndex, row := range rows {
			pieces := []string{strconv.Itoa(rowIndex)}
			complete := true
			for keyIndex, sourceIndex := range sourceIndexes {
				if row.Cells[sourceIndex].IsNull {
					complete = false
					break
				}
				arguments = append(arguments, row.Cells[sourceIndex].Value)
				pieces = append(pieces, typeExpression(targetKeys[keyIndex], len(arguments)))
			}
			if !complete {
				arguments = arguments[:len(arguments)-(len(pieces)-1)]
				continue
			}
			wantedRows = append(wantedRows, "("+strings.Join(pieces, ", ")+")")
			if len(wantedRows) >= MaximumPeeks {
				break
			}
		}
		if len(wantedRows) == 0 {
			continue
		}
		wantedNames := []string{"row_index"}
		joins := make([]string, 0)
		for index, target := range targetKeys {
			name := "key" + strconv.Itoa(index)
			wantedNames = append(wantedNames, name)
			joins = append(joins, "target."+quoteIdentifier(target.Name)+" = wanted."+name)
		}
		selected := []string{"wanted.row_index::text"}
		targetNames := make([]string, 0, len(targets))
		for _, target := range targets {
			selected = append(selected, "target."+quoteIdentifier(target.Name)+"::text")
			targetNames = append(targetNames, target.Name)
		}
		statement := "WITH wanted (" + strings.Join(wantedNames, ", ") + ") AS (VALUES " + strings.Join(wantedRows, ", ") + ") SELECT " + strings.Join(selected, ", ") + " FROM wanted JOIN " + qualifiedRelation(constraint.ReferenceSchema, constraint.ReferenceTable) + " AS target ON " + strings.Join(joins, " AND ") + fmt.Sprintf(" LIMIT %d", MaximumPeeks)
		result, queryError := engines.Query(databaseContext(requestContext, record, engines.ReadOnly), record.Name, statement, arguments...)
		if queryError != nil {
			continue
		}
		for _, values := range result.Rows {
			if len(values) != len(targets)+1 {
				continue
			}
			rowIndex, parseError := strconv.Atoi(fmt.Sprint(values[0]))
			if parseError != nil || rowIndex < 0 || rowIndex >= len(rows) {
				continue
			}
			peek := &PeekView{Table: constraint.ReferenceSchema + "." + constraint.ReferenceTable, Columns: targetNames}
			for _, value := range values[1:] {
				peek.Cells = append(peek.Cells, peekCell(value))
			}
			for _, index := range sourceIndexes {
				rows[rowIndex].Cells[index].Peek = peek
			}
		}
	}
}
