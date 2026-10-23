package explorer

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"sqldash/engines"
	"sqldash/models"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

type ConstraintView struct {
	Name       string
	Kind       string
	Columns    []string
	Definition string
}

func relationQuery(schema string, table string) string {
	return url.Values{"schema": {schema}, "table": {table}}.Encode()
}

func qualifiedRelation(schema string, table string) string {
	return quoteIdentifier(schema) + "." + quoteIdentifier(table)
}

func systemPostgresSchema(name string) bool {
	return name == "pg_catalog" || name == "information_schema"
}

func mutablePostgresSchema(name string) bool {
	lowered := strings.ToLower(name)
	return lowered != "" && !strings.HasPrefix(lowered, "pg_") && lowered != "information_schema" && lowered != "sqd_control" && lowered != "sqd_extensions"
}

func visiblePostgresSchema(name string) bool {
	return systemPostgresSchema(name) || mutablePostgresSchema(name)
}

func postgresSchemaChoice(name string) ChoiceView {
	label := name
	if systemPostgresSchema(name) {
		label += " (read-only)"
	}
	return ChoiceView{Value: name, Label: label}
}

func postgresRelations(requestContext context.Context, record *models.Database, schema string) ([]engines.Relation, *fiber.Error) {
	if schema != "" && !visiblePostgresSchema(schema) {
		return nil, shortcuts.ServiceError(http.StatusNotFound, TableMissing)
	}
	catalog, catalogError := catalogFor(record)
	if catalogError != nil {
		return nil, catalogError
	}
	relations, readError := catalog.Relations(databaseContext(requestContext, record, engines.ReadOnly), record, schema)
	if readError != nil {
		return nil, shortcuts.ServiceError(http.StatusBadGateway, TablesUnavailable)
	}
	visible := make([]engines.Relation, 0, len(relations))
	for _, relation := range relations {
		if visiblePostgresSchema(relation.Schema) {
			visible = append(visible, relation)
		}
	}
	return visible, nil
}

func tableViews(relations []engines.Relation, schema string) []TableView {
	views := make([]TableView, 0, len(relations))
	for _, relation := range relations {
		if schema != "" && relation.Schema != schema {
			continue
		}
		views = append(views, TableView{Name: relation.Name, Schema: relation.Schema, Kind: relation.Kind,
			IsView: relation.Kind != "table" && relation.Kind != "partitioned table", Rows: relation.Rows,
			Query: relationQuery(relation.Schema, relation.Name)})
	}
	return views
}

func schemaChoices(relations []engines.Relation) []ChoiceView {
	seen := map[string]bool{}
	for _, relation := range relations {
		if !visiblePostgresSchema(relation.Schema) {
			continue
		}
		seen[relation.Schema] = true
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	choices := make([]ChoiceView, 0, len(names))
	for _, name := range names {
		choices = append(choices, postgresSchemaChoice(name))
	}
	return choices
}

func postgresCompletionTables(requestContext context.Context, record *models.Database) ([]SchemaTable, *fiber.Error) {
	catalog, catalogError := catalogFor(record)
	if catalogError != nil {
		return nil, catalogError
	}
	readContext := databaseContext(requestContext, record, engines.ReadOnly)
	if provider, supported := catalog.(interface {
		RelationColumns(context.Context, *models.Database) ([]engines.RelationColumns, error)
	}); supported {
		relations, readError := provider.RelationColumns(readContext, record)
		if readError != nil {
			return nil, shortcuts.ServiceError(http.StatusBadGateway, TablesUnavailable)
		}
		tables := make([]SchemaTable, 0, len(relations))
		for _, relation := range relations {
			if visiblePostgresSchema(relation.Schema) {
				columns := relation.Columns
				if columns == nil {
					columns = []string{}
				}
				tables = append(tables, SchemaTable{Schema: relation.Schema, Name: relation.Name, Kind: relation.Kind, Columns: columns})
			}
		}
		return tables, nil
	}
	relations, relationsError := postgresRelations(requestContext, record, "")
	if relationsError != nil {
		return nil, relationsError
	}
	tables := make([]SchemaTable, 0, len(relations))
	for _, relation := range relations {
		columns, columnsError := catalog.Columns(readContext, record, relation.Schema, relation.Name)
		if columnsError != nil {
			continue
		}
		table := SchemaTable{Schema: relation.Schema, Name: relation.Name, Kind: relation.Kind, Columns: make([]string, 0, len(columns))}
		for _, column := range columns {
			table.Columns = append(table.Columns, column.Name)
		}
		tables = append(tables, table)
	}
	return tables, nil
}

func postgresTables(requestContext context.Context, record *models.Database, schema string) ([]TableView, *fiber.Error) {
	relations, relationsError := postgresRelations(requestContext, record, schema)
	if relationsError != nil {
		return nil, relationsError
	}
	return tableViews(relations, schema), nil
}

func postgresColumnViews(columns []engines.Column) []ColumnView {
	views := make([]ColumnView, 0, len(columns))
	for _, column := range columns {
		view := ColumnView{Name: column.Name, Kind: column.Kind, NotNull: column.NotNull,
			PrimaryKey: column.PrimaryKey, PrimaryPosition: column.PrimaryPosition, Default: column.Default,
			Generated: column.Generated, Identity: column.Identity, ReadOnly: column.Generated || column.Identity,
			ReferenceSchema: column.ReferenceSchema}
		if len(column.ReferenceColumns) == 1 {
			view.References = column.References
			view.OnColumn = column.ReferenceColumns[0]
		}
		views = append(views, view)
	}
	return views
}

func postgresColumns(requestContext context.Context, record *models.Database, schema string, table string) ([]ColumnView, *fiber.Error) {
	relations, relationsError := postgresTables(requestContext, record, schema)
	if relationsError != nil {
		return nil, relationsError
	}
	if !known(relations, table) {
		return nil, shortcuts.ServiceError(http.StatusNotFound, TableMissing)
	}
	catalog, catalogError := catalogFor(record)
	if catalogError != nil {
		return nil, catalogError
	}
	columns, readError := catalog.Columns(databaseContext(requestContext, record, engines.ReadOnly), record, schema, table)
	if readError != nil {
		return nil, shortcuts.ServiceError(http.StatusBadGateway, TablesUnavailable)
	}
	return postgresColumnViews(columns), nil
}

func postgresDefinition(requestContext context.Context, record *models.Database, schema string, table string) (*engines.Definition, *fiber.Error) {
	tables, tablesError := postgresTables(requestContext, record, schema)
	if tablesError != nil {
		return nil, tablesError
	}
	if !known(tables, table) {
		return nil, shortcuts.ServiceError(http.StatusNotFound, TableMissing)
	}
	catalog, catalogError := catalogFor(record)
	if catalogError != nil {
		return nil, catalogError
	}
	definition, readError := catalog.Structure(databaseContext(requestContext, record, engines.ReadOnly), record, schema, table)
	if readError != nil || definition == nil {
		return nil, shortcuts.ServiceError(http.StatusBadGateway, TablesUnavailable)
	}
	return definition, nil
}

func writableRelation(kind string) bool {
	return kind == "table" || kind == "partitioned table"
}

func primaryColumns(columns []ColumnView) []ColumnView {
	keys := make([]ColumnView, 0)
	for _, column := range columns {
		if column.PrimaryKey {
			keys = append(keys, column)
		}
	}
	sort.SliceStable(keys, func(first int, second int) bool { return keys[first].PrimaryPosition < keys[second].PrimaryPosition })
	return keys
}

func columnByName(columns []ColumnView, name string) (ColumnView, bool) {
	for _, column := range columns {
		if column.Name == name {
			return column, true
		}
	}
	return ColumnView{}, false
}

func referenceLinks(constraints []engines.Constraint, cells []CellView, columns []ColumnView) {
	for _, constraint := range constraints {
		if !visiblePostgresSchema(constraint.ReferenceSchema) || constraint.ReferenceTable == "" || len(constraint.Columns) == 0 || len(constraint.Columns) != len(constraint.ReferenceColumns) {
			continue
		}
		query := url.Values{"schema": {constraint.ReferenceSchema}, "table": {constraint.ReferenceTable}}
		indexes := make([]int, 0, len(constraint.Columns))
		for index, name := range constraint.Columns {
			for columnIndex, column := range columns {
				if column.Name == name && !cells[columnIndex].IsNull {
					indexes = append(indexes, columnIndex)
					query.Add("column", constraint.ReferenceColumns[index])
					query.Add("operator", "is")
					query.Add("value", cells[columnIndex].Value)
				}
			}
		}
		if len(indexes) != len(constraint.Columns) {
			continue
		}
		for _, index := range indexes {
			cells[index].LinkTo = constraint.ReferenceTable
			cells[index].LinkQuery = query.Encode()
		}
	}
}

func typeExpression(column ColumnView, position int) string {
	return parameterAt(position) + "::text::" + column.Kind
}

func matchesQualifiedName(catalogName string, schema string, name string) bool {
	if catalogName == name {
		return true
	}
	parts := make([]string, 0, 2)
	var held strings.Builder
	quoted := false
	for index := 0; index < len(catalogName); index++ {
		switch catalogName[index] {
		case '"':
			if quoted && index+1 < len(catalogName) && catalogName[index+1] == '"' {
				held.WriteByte('"')
				index++
			} else {
				quoted = !quoted
			}
		case '.':
			if quoted {
				held.WriteByte('.')
			} else {
				parts = append(parts, held.String())
				held.Reset()
			}
		default:
			held.WriteByte(catalogName[index])
		}
	}
	parts = append(parts, held.String())
	return !quoted && len(parts) == 2 && parts[0] == schema && parts[1] == name
}

func safeObjectName(name string) bool {
	return strings.TrimSpace(name) != "" && len(name) <= 63 && !strings.ContainsRune(name, 0)
}
