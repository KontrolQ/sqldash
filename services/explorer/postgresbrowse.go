package explorer

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"sqldash/config"
	"sqldash/engines"
	"sqldash/models"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func browsePostgres(requestContext context.Context, record *models.Database, asked Ask) (*BrowseContext, *fiber.Error) {
	schema := selectedSchema([]string{asked.Schema})
	if !visiblePostgresSchema(schema) {
		return nil, shortcuts.ServiceError(http.StatusNotFound, TableMissing)
	}
	relations, catalogError := postgresRelations(requestContext, record, "")
	if catalogError != nil {
		return nil, catalogError
	}
	schemas, schemasError := postgresSchemas(requestContext, record, relations)
	if schemasError != nil {
		return nil, schemasError
	}
	shown := &BrowseContext{Title: record.Name, Database: record.Name, Engine: record.Kind(), Schema: schema,
		Address: config.PostgresHost(), Tables: tableViews(relations, schema), Schemas: schemas,
		Page: max(asked.Page, 1), PageSize: DefaultPageSize, BrowsePath: DatabasePath + url.PathEscape(record.Name) + BrowseSuffix,
		StructurePath:  DatabasePath + url.PathEscape(record.Name) + StructureSuffix,
		ReadOnlySchema: systemPostgresSchema(schema)}
	if asked.Table == "" {
		return shown, nil
	}
	if !known(shown.Tables, asked.Table) {
		return nil, shortcuts.ServiceError(http.StatusNotFound, TableMissing)
	}
	shown.Table = asked.Table
	shown.Title = schema + "." + asked.Table
	shown.RelationQuery = relationQuery(schema, asked.Table)
	shown.Carry = shown.RelationQuery
	shown.View = shown.RelationQuery
	definition, definitionError := postgresDefinition(requestContext, record, schema, asked.Table)
	if definitionError != nil {
		if shown.ReadOnlySchema && definitionError.Code == http.StatusBadGateway {
			shown.Unavailable = CatalogUnavailable
			return shown, nil
		}
		return nil, definitionError
	}
	shown.IsView = !writableRelation(definition.Relation.Kind)
	shown.Columns = postgresColumnViews(definition.Columns)
	shown.Insertable = managedSchema(schema) && !shown.IsView && CanWrite(requestContext, record.Name)
	shown.Editable = shown.Insertable && len(primaryColumns(shown.Columns)) > 0
	shown.Search = asked.Search
	shown.Direction = directionOf(asked.Direction)
	if columnNamed(shown.Columns, asked.Sort) {
		shown.Sort = asked.Sort
	}
	shown.Filters = postgresFilters(shown.Columns, asked.Filters)
	shown.Operators = OperatorChoices()
	shown.ColumnChoices = columnChoices(shown.Columns)
	shown.Carry = carrySchema(schema, carryOf(asked.Table, asked.Search, shown.Filters))
	shown.CarryNoSearch = carrySchema(schema, withoutSearch(asked.Table, shown.Filters))
	shown.View = viewOf(shown)
	for index := range shown.Filters {
		shown.Filters[index].RemoveURL = shown.BrowsePath + "?" + carrySchema(schema, withoutFilter(asked.Table, asked.Search, shown.Filters, index))
	}
	where, arguments := postgresWhere(shown.Columns, asked.Search, shown.Filters)
	readContext := databaseContext(requestContext, record, engines.ReadOnly)
	count, readError := engines.Query(readContext, record.Name, "SELECT count(*) FROM "+qualifiedRelation(schema, asked.Table)+where, arguments...)
	if readError != nil || len(count.Rows) != 1 || len(count.Rows[0]) != 1 {
		return unavailablePostgresBrowse(shown)
	}
	total, parseError := strconv.ParseInt(fmt.Sprint(count.Rows[0][0]), 10, 64)
	if parseError != nil {
		return unavailablePostgresBrowse(shown)
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
	selected := make([]string, 0, len(shown.Columns)+1)
	for _, column := range shown.Columns {
		selected = append(selected, quoteIdentifier(column.Name)+"::text")
	}
	stable := managedSchema(schema) && !shown.IsView && len(primaryColumns(shown.Columns)) > 0
	if stable {
		selected = append(selected, "xmin::text")
	}
	statement := "SELECT " + strings.Join(selected, ", ") + " FROM " + qualifiedRelation(schema, asked.Table) + where
	order := make([]string, 0)
	if shown.Sort != "" {
		column, _ := columnByName(shown.Columns, shown.Sort)
		ordered := quoteIdentifier(column.Name)
		if column.Kind == "json" {
			ordered += "::jsonb"
		}
		if column.Kind == "json[]" {
			ordered += "::jsonb[]"
		}
		order = append(order, ordered+" "+strings.ToUpper(shown.Direction))
	}
	for _, key := range primaryColumns(shown.Columns) {
		if key.Name != shown.Sort {
			order = append(order, quoteIdentifier(key.Name))
		}
	}
	if len(order) > 0 {
		statement += " ORDER BY " + strings.Join(order, ", ")
	}
	statement += fmt.Sprintf(" LIMIT %d OFFSET %d", shown.PageSize, (shown.Page-1)*shown.PageSize)
	result, queryError := engines.Query(readContext, record.Name, statement, arguments...)
	if queryError != nil {
		return unavailablePostgresBrowse(shown)
	}
	shown.Duration = fmt.Sprintf(DurationFormat, result.DurationMs)
	for _, values := range result.Rows {
		if len(values) < len(shown.Columns) {
			return unavailablePostgresBrowse(shown)
		}
		row := RowView{Cells: make([]CellView, 0, len(shown.Columns))}
		for index, column := range shown.Columns {
			cell := cellFor(values[index], column)
			cell.Value = textOr(values[index], "")
			cell.LinkTo = ""
			cell.LinkColumn = ""
			row.Cells = append(row.Cells, cell)
		}
		if stable {
			if len(values) != len(shown.Columns)+1 {
				return unavailablePostgresBrowse(shown)
			}
			identity, identityError := signRow(record.ID, schema, asked.Table, shown.Columns, values, fmt.Sprint(values[len(shown.Columns)]))
			if identityError != nil {
				return nil, shortcuts.ServiceError(http.StatusInternalServerError, "Row identities are unavailable.")
			}
			row.Key = identity
		}
		referenceLinks(definition.Constraints, row.Cells, shown.Columns)
		shown.Rows = append(shown.Rows, row)
	}
	attachPostgresPeeks(requestContext, record, definition.Constraints, shown.Columns, shown.Rows)
	if gridError := prepareGrid(shown, "postgres"); gridError != nil {
		return nil, gridError
	}
	shown.View = viewOf(shown)
	shown.ColumnNames = namesOf(shown.Columns)
	if total > 0 {
		shown.FirstRow = (shown.Page-1)*shown.PageSize + 1
		shown.LastRow = shown.FirstRow + len(shown.Rows) - 1
	}
	return shown, nil
}

func unavailablePostgresBrowse(shown *BrowseContext) (*BrowseContext, *fiber.Error) {
	if !shown.ReadOnlySchema {
		return nil, shortcuts.ServiceError(http.StatusBadGateway, RowsUnavailable)
	}
	shown.Unavailable = CatalogRestricted
	shown.Rows = nil
	shown.Total = 0
	shown.Insertable = false
	shown.Editable = false
	return shown, nil
}

func carrySchema(schema string, query string) string {
	values, _ := url.ParseQuery(query)
	values.Set("schema", schema)
	return values.Encode()
}

func parameterAt(position int) string { return "$" + strconv.Itoa(position) }

func postgresFilters(columns []ColumnView, asked []Filter) []FilterView {
	kept := make([]FilterView, 0, len(asked))
	for _, filter := range asked {
		if !columnNamed(columns, filter.Column) || !knownOperator(filter.Operator) {
			continue
		}
		kept = append(kept, FilterView{Column: filter.Column, Operator: filter.Operator, Value: filter.Value, Label: LabelForOperator(filter.Operator)})
	}
	return kept
}

func postgresWhere(columns []ColumnView, search string, filters []FilterView) (string, []any) {
	pieces := make([]string, 0)
	arguments := make([]any, 0)
	if strings.TrimSpace(search) != "" {
		arguments = append(arguments, "%"+search+"%")
		searches := make([]string, 0, len(columns))
		for _, column := range columns {
			searches = append(searches, quoteIdentifier(column.Name)+"::text ILIKE $1")
		}
		if len(searches) > 0 {
			pieces = append(pieces, "("+strings.Join(searches, " OR ")+")")
		}
	}
	for _, filter := range filters {
		column, exists := columnByName(columns, filter.Column)
		if !exists {
			continue
		}
		name := quoteIdentifier(column.Name)
		if filter.Operator == OperatorEmpty {
			pieces = append(pieces, name+" IS NULL")
			continue
		}
		if filter.Operator == OperatorFilled {
			pieces = append(pieces, name+" IS NOT NULL")
			continue
		}
		value := filter.Value
		comparison := "="
		left := name
		cast := column.Kind
		switch filter.Operator {
		case OperatorContains:
			left += "::text"
			comparison = "ILIKE"
			value = "%" + value + "%"
			cast = "text"
		case OperatorMisses:
			left += "::text"
			comparison = "NOT ILIKE"
			value = "%" + value + "%"
			cast = "text"
		case OperatorStarts:
			left += "::text"
			comparison = "ILIKE"
			value += "%"
			cast = "text"
		case OperatorEnds:
			left += "::text"
			comparison = "ILIKE"
			value = "%" + value
			cast = "text"
		case OperatorIsNot:
			comparison = "<>"
		case OperatorAbove:
			comparison = ">"
		case OperatorAtLeast:
			comparison = ">="
		case OperatorBelow:
			comparison = "<"
		case OperatorAtMost:
			comparison = "<="
		}
		if cast != "text" && column.Kind == "json" {
			left += "::jsonb"
			cast = "jsonb"
		}
		if cast != "text" && column.Kind == "json[]" {
			left += "::jsonb[]"
			cast = "jsonb[]"
		}
		arguments = append(arguments, value)
		pieces = append(pieces, left+" "+comparison+" "+parameterAt(len(arguments))+"::text::"+cast)
	}
	if len(pieces) == 0 {
		return "", arguments
	}
	return " WHERE " + strings.Join(pieces, " AND "), arguments
}
