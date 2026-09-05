package explorer

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"sqldash/config"
	"sqldash/sqld"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

type Ask struct {
	Database  string
	Table     string
	Page      int
	Sort      string
	Direction string
	Search    string
	Filters   []Filter
}

func Browse(requestContext context.Context, asked Ask) (*BrowseContext, *fiber.Error) {
	tables, tablesError := Tables(requestContext, asked.Database)
	if tablesError != nil {
		return nil, tablesError
	}

	shown := &BrowseContext{
		Title:         asked.Database,
		Database:      asked.Database,
		Address:       asked.Database + "." + config.Server.Domain,
		Tables:        tables,
		PageSize:      DefaultPageSize,
		Page:          1,
		BrowsePath:    DatabasePath + asked.Database + BrowseSuffix,
		StructurePath: DatabasePath + asked.Database + StructureSuffix,
	}

	if asked.Table == "" {
		return shown, nil
	}

	if !known(tables, asked.Table) {
		return nil, shortcuts.ServiceError(http.StatusNotFound, TableMissing)
	}

	shown.Table = asked.Table
	shown.Title = asked.Table
	shown.Search = asked.Search
	shown.Direction = directionOf(asked.Direction)

	for _, table := range tables {
		if table.Name == asked.Table {
			shown.IsView = table.IsView
		}
	}

	columns, columnsError := Columns(requestContext, asked.Database, asked.Table)
	if columnsError != nil {
		return nil, columnsError
	}

	shown.Columns = columns
	shown.KeyColumn = keyColumnOf(requestContext, asked.Database, asked.Table, columns, shown.IsView)
	shown.Editable = shown.KeyColumn != ""

	if asked.Sort != "" && columnNamed(columns, asked.Sort) {
		shown.Sort = asked.Sort
	}

	shown.Filters = keptFilters(columns, asked.Filters)
	shown.Operators = OperatorChoices()
	shown.ColumnChoices = columnChoices(columns)
	shown.Carry = carryOf(shown.Table, shown.Search, shown.Filters)

	for index := range shown.Filters {
		shown.Filters[index].RemoveURL = shown.BrowsePath + "?" +
			withoutFilter(shown.Table, shown.Search, shown.Filters, index)
	}

	where, arguments := whereClause(columns, asked.Search, shown.Filters)

	total, countError := countRows(requestContext, asked.Database, asked.Table, where, arguments)
	if countError != nil {
		return nil, countError
	}

	shown.Total = total
	shown.Pages = pagesFor(total, DefaultPageSize)

	shown.Page = asked.Page
	if shown.Page < 1 {
		shown.Page = 1
	}

	if shown.Pages > 0 && shown.Page > shown.Pages {
		shown.Page = shown.Pages
	}

	shown.PreviousPage = shown.Page - 1
	shown.NextPage = shown.Page + 1

	rows, rowsError := readRows(requestContext, asked.Database, asked.Table, columns, shown, where, arguments)
	if rowsError != nil {
		return nil, rowsError
	}

	shown.Rows = rows
	shown.ColumnNames = namesOf(columns)
	shown.View = viewOf(shown)

	if total > 0 {
		shown.FirstRow = (shown.Page-1)*shown.PageSize + 1
		shown.LastRow = shown.FirstRow + len(rows) - 1
	}

	return shown, nil
}

func readRows(requestContext context.Context, databaseName string, table string, columns []ColumnView, shown *BrowseContext, where string, arguments []any) ([]RowView, *fiber.Error) {
	selected := make([]string, 0, len(columns)+1)

	if shown.KeyColumn != "" {
		selected = append(selected, quoteIdentifier(shown.KeyColumn))
	}

	for _, column := range columns {
		selected = append(selected, quoteIdentifier(column.Name))
	}

	statement := "SELECT " + strings.Join(selected, ", ") + " FROM " + quoteIdentifier(table) + where

	if shown.Sort != "" {
		statement += " ORDER BY " + quoteIdentifier(shown.Sort) + " " + strings.ToUpper(shown.Direction)
	}

	statement += fmt.Sprintf(" LIMIT %d OFFSET %d", shown.PageSize, (shown.Page-1)*shown.PageSize)

	held, queryError := sqld.Query(requestContext, databaseName, statement, arguments...)
	if queryError != nil {
		logger.Errorf(LogPrefix, ReadFailedLog, databaseName, queryError)
		return nil, shortcuts.ServiceError(http.StatusBadGateway, RowsUnavailable)
	}

	shown.Duration = fmt.Sprintf(DurationFormat, held.DurationMs)

	offset := 0
	if shown.KeyColumn != "" {
		offset = 1
	}

	views := make([]RowView, 0, len(held.Rows))

	for _, row := range held.Rows {
		view := RowView{Cells: make([]CellView, 0, len(columns))}

		if shown.KeyColumn != "" {
			view.Key = fmt.Sprint(row[0])
		}

		for index, column := range columns {
			view.Cells = append(view.Cells, cellFor(row[index+offset], column))
		}

		views = append(views, view)
	}

	return views, nil
}

func cellFor(value any, column ColumnView) CellView {
	if value == nil {
		return CellView{Column: column.Name, IsNull: true}
	}

	text := fmt.Sprint(value)

	if held, isBytes := value.([]byte); isBytes {
		text = fmt.Sprintf("%d bytes", len(held))
	}

	cell := CellView{Column: column.Name, Text: text}

	if len(text) > MaximumCellText {
		cell.Text = text[:MaximumCellText]
		cell.Truncated = true
	}

	if column.References != "" && text != "" {
		cell.LinkTo = column.References
		cell.LinkColumn = column.OnColumn
	}

	return cell
}

func countRows(requestContext context.Context, databaseName string, table string, where string, arguments []any) (int64, *fiber.Error) {
	held, queryError := sqld.Query(
		requestContext, databaseName,
		fmt.Sprintf(CountSQL, quoteIdentifier(table))+where, arguments...,
	)

	if queryError != nil {
		logger.Errorf(LogPrefix, ReadFailedLog, databaseName, queryError)
		return 0, shortcuts.ServiceError(http.StatusBadGateway, RowsUnavailable)
	}

	if len(held.Rows) == 0 {
		return 0, nil
	}

	if count, isNumber := held.Rows[0][0].(int64); isNumber {
		return count, nil
	}

	return 0, nil
}

func keyColumnOf(requestContext context.Context, databaseName string, table string, columns []ColumnView, isView bool) string {
	if isView {
		return ""
	}

	keys := make([]ColumnView, 0)

	for _, column := range columns {
		if column.PrimaryKey {
			keys = append(keys, column)
		}
	}

	if len(keys) == 1 {
		return keys[0].Name
	}

	if _, queryError := sqld.Query(requestContext, databaseName, fmt.Sprintf(HasRowIdSQL, quoteIdentifier(table))); queryError == nil {
		return RowIdentifier
	}

	return ""
}

func columnNamed(columns []ColumnView, name string) bool {
	for _, column := range columns {
		if column.Name == name {
			return true
		}
	}

	return false
}

func pagesFor(total int64, size int) int {
	if total == 0 {
		return 0
	}

	pages := int(total) / size
	if int(total)%size != 0 {
		pages++
	}

	return pages
}

func namesOf(columns []ColumnView) string {
	names := make([]string, 0, len(columns))

	for _, column := range columns {
		names = append(names, column.Name)
	}

	return strings.Join(names, ",")
}
