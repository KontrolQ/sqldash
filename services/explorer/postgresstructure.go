package explorer

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"sqldash/config"
	"sqldash/engines"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/services/access"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func structurePostgres(requestContext context.Context, record *models.Database, schema string, table string) (*StructureContext, *fiber.Error) {
	if !visiblePostgresSchema(schema) {
		return nil, shortcuts.ServiceError(http.StatusNotFound, TableMissing)
	}
	relations, relationsError := postgresRelations(requestContext, record, "")
	if relationsError != nil {
		return nil, relationsError
	}
	choices, choicesError := postgresSchemas(requestContext, record, relations)
	if choicesError != nil {
		return nil, choicesError
	}
	shown := &StructureContext{Title: record.Name, Database: record.Name, Address: config.PostgresHost(), Engine: record.Kind(),
		Schema: schema, Schemas: choices, Tables: tableViews(relations, schema), Table: table, RelationQuery: relationQuery(schema, table),
		BrowsePath: DatabasePath + url.PathEscape(record.Name) + BrowseSuffix, StructurePath: DatabasePath + url.PathEscape(record.Name) + StructureSuffix,
		ObjectPath: DatabasePath + url.PathEscape(record.Name) + StructureSuffix + "/object", Kinds: postgresTypeChoices(),
		ReadOnlySchema: systemPostgresSchema(schema)}
	for _, choice := range choices {
		if managedSchema(choice.Value) {
			shown.MutableSchemas = append(shown.MutableSchemas, choice)
		}
	}
	if table != "" {
		shown.Title = schema + "." + table
	}
	actor := access.FromContext(requestContext)
	if actor != nil {
		fresh, loadError := access.Load(actor.Account.ID)
		shown.CanManageObjects = managedSchema(schema) && loadError == nil && fresh.Can(permissions.PostgresObjectsManage)
	}
	catalog, catalogError := catalogFor(record)
	if catalogError != nil {
		return nil, catalogError
	}
	if table != "" && !known(shown.Tables, table) {
		return nil, shortcuts.ServiceError(http.StatusNotFound, TableMissing)
	}
	definition, readError := catalog.Structure(databaseContext(requestContext, record, engines.ReadOnly), record, schema, table)
	if readError != nil || definition == nil {
		if shown.ReadOnlySchema {
			shown.Unavailable = CatalogUnavailable
			return shown, nil
		}
		return nil, shortcuts.ServiceError(http.StatusBadGateway, TablesUnavailable)
	}
	shown.IsView = table != "" && !writableRelation(definition.Relation.Kind)
	shown.Columns = postgresColumnViews(definition.Columns)
	shown.Definition = definition.SQL
	for _, index := range definition.Indexes {
		origin := CreatedLabel
		if index.Primary {
			origin = PrimaryKeyLabel
		}
		shown.Indexes = append(shown.Indexes, IndexView{Name: index.Name, Unique: index.Unique, Columns: index.Columns, Origin: origin})
	}
	for _, constraint := range definition.Constraints {
		shown.Constraints = append(shown.Constraints, ConstraintView{Name: constraint.Name, Kind: constraint.Kind, Columns: constraint.Columns, Definition: constraint.Definition})
		if constraint.ReferenceTable != "" {
			shown.Relations = append(shown.Relations, RelationView{FromTable: table, FromColumn: strings.Join(constraint.Columns, ", "), ToTable: constraint.ReferenceSchema + "." + constraint.ReferenceTable, ToColumn: strings.Join(constraint.ReferenceColumns, ", ")})
		}
	}
	for _, trigger := range definition.Triggers {
		shown.Triggers = append(shown.Triggers, TriggerView{Name: trigger.Name, Definition: trigger.Definition})
	}
	for _, function := range definition.Functions {
		shown.Functions = append(shown.Functions, TriggerView{Name: function.Name, Definition: function.Definition})
	}
	for _, sequence := range definition.Sequences {
		shown.Sequences = append(shown.Sequences, TriggerView{Name: sequence.Name, Definition: sequence.Definition})
	}
	return shown, nil
}

func postgresSchemas(requestContext context.Context, record *models.Database, relations []engines.Relation) ([]ChoiceView, *fiber.Error) {
	catalog, catalogError := catalogFor(record)
	if catalogError != nil {
		return nil, catalogError
	}
	provider, supported := catalog.(interface {
		Schemas(context.Context, *models.Database) ([]string, error)
	})
	if !supported {
		return schemaChoices(relations), nil
	}
	schemas, readError := provider.Schemas(databaseContext(requestContext, record, engines.ReadOnly), record)
	if readError != nil {
		return nil, shortcuts.ServiceError(http.StatusBadGateway, TablesUnavailable)
	}
	choices := make([]ChoiceView, 0, len(schemas))
	for _, schema := range schemas {
		if !visiblePostgresSchema(schema) {
			continue
		}
		choices = append(choices, postgresSchemaChoice(schema))
	}
	return choices, nil
}
