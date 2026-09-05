package explorer

import (
	"context"
	"fmt"
	"net/http"

	"sqldash/config"
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
		Address:       databaseName + "." + config.Server.Domain,
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
	shown.Triggers = triggersOf(requestContext, databaseName, table)
	shown.Relations = relationsOf(requestContext, databaseName, table)
	shown.UsedBy = pointingAt(requestContext, databaseName, tables, table)
	shown.Definition = definitionOf(requestContext, databaseName, table)

	return shown, nil
}

func triggersOf(requestContext context.Context, databaseName string, table string) []TriggerView {
	held, queryError := sqld.Query(requestContext, databaseName, TriggerSQL, table)
	if queryError != nil {
		return nil
	}

	views := make([]TriggerView, 0, len(held.Rows))

	for _, row := range held.Rows {
		view := TriggerView{Name: fmt.Sprint(row[0])}

		if row[1] != nil {
			view.Definition = fmt.Sprint(row[1])
		}

		views = append(views, view)
	}

	return views
}

func relationsOf(requestContext context.Context, databaseName string, table string) []RelationView {
	held, queryError := sqld.Query(requestContext, databaseName, fmt.Sprintf(ForeignKeyListSQL, quoteIdentifier(table)))
	if queryError != nil {
		return nil
	}

	views := make([]RelationView, 0, len(held.Rows))

	for _, row := range held.Rows {
		views = append(views, RelationView{
			FromTable:  table,
			FromColumn: fmt.Sprint(row[3]),
			ToTable:    fmt.Sprint(row[2]),
			ToColumn:   textOr(row[4], ""),
			OnUpdate:   textOr(row[5], NoAction),
			OnDelete:   textOr(row[6], NoAction),
		})
	}

	return views
}

func pointingAt(requestContext context.Context, databaseName string, tables []TableView, table string) []RelationView {
	views := make([]RelationView, 0)

	for _, one := range tables {
		if one.IsView || one.Name == table {
			continue
		}

		for _, relation := range relationsOf(requestContext, databaseName, one.Name) {
			if relation.ToTable == table {
				views = append(views, relation)
			}
		}
	}

	return views
}

func textOr(value any, fallback string) string {
	if value == nil {
		return fallback
	}

	return fmt.Sprint(value)
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
			Origin: originLabel(fmt.Sprint(row[3])),
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

func originLabel(origin string) string {
	switch origin {
	case CreatedOrigin:
		return CreatedLabel
	case UniqueOrigin:
		return UniqueLabel
	case PrimaryKeyOrigin:
		return PrimaryKeyLabel
	}

	return origin
}
