package explorer

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"sqldash/engines"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

type FieldValue struct {
	State string
	Value string
}

func InsertFields(requestContext context.Context, databaseName string, schema string, table string, fields map[string]FieldValue) *fiber.Error {
	record, permissionError := authorizedDatabase(requestContext, databaseName, permissions.DataWrite)
	if permissionError != nil {
		return permissionError
	}
	if record.Kind() != models.EnginePostgres {
		values := make(map[string]string, len(fields))
		for name, field := range fields {
			values[name] = field.Value
		}
		return InsertRow(requestContext, databaseName, table, values)
	}
	return insertPostgres(requestContext, record, selectedSchema([]string{schema}), table, fields)
}

func mutationColumns(requestContext context.Context, record *models.Database, schema string, table string, requireKey bool) ([]ColumnView, *fiber.Error) {
	if !managedSchema(schema) {
		return nil, shortcuts.ServiceError(http.StatusForbidden, ReadOnlySchema)
	}
	definition, definitionError := postgresDefinition(requestContext, record, schema, table)
	if definitionError != nil {
		return nil, definitionError
	}
	columns := postgresColumnViews(definition.Columns)
	if !writableRelation(definition.Relation.Kind) || (requireKey && len(primaryColumns(columns)) == 0) {
		return nil, shortcuts.ServiceError(http.StatusBadRequest, NotEditable)
	}
	return columns, nil
}

func fieldExpression(field FieldValue, column ColumnView, arguments *[]any) (string, *fiber.Error) {
	switch field.State {
	case "default":
		return "DEFAULT", nil
	case "null":
		if column.NotNull {
			return "", shortcuts.ServiceError(http.StatusBadRequest, "This column cannot be NULL.")
		}
		return "NULL", nil
	case "value", "":
		*arguments = append(*arguments, field.Value)
		return typeExpression(column, len(*arguments)), nil
	default:
		return "", shortcuts.ServiceError(http.StatusBadRequest, "Unknown field state.")
	}
}

func insertPostgres(requestContext context.Context, record *models.Database, schema string, table string, fields map[string]FieldValue) *fiber.Error {
	columns, columnsError := mutationColumns(requestContext, record, schema, table, false)
	if columnsError != nil {
		return columnsError
	}
	for name := range fields {
		if !columnNamed(columns, name) {
			return shortcuts.ServiceError(http.StatusBadRequest, ColumnMissing)
		}
	}
	names := make([]string, 0)
	expressions := make([]string, 0)
	arguments := make([]any, 0)
	for _, column := range columns {
		field, given := fields[column.Name]
		if !given || field.State == "default" {
			continue
		}
		if column.ReadOnly {
			return shortcuts.ServiceError(http.StatusBadRequest, "Generated and identity columns are read-only.")
		}
		expression, fieldError := fieldExpression(field, column, &arguments)
		if fieldError != nil {
			return fieldError
		}
		names = append(names, quoteIdentifier(column.Name))
		expressions = append(expressions, expression)
	}
	statement := "INSERT INTO " + qualifiedRelation(schema, table)
	if len(names) == 0 {
		statement += " DEFAULT VALUES"
	} else {
		statement += " (" + strings.Join(names, ", ") + ") VALUES (" + strings.Join(expressions, ", ") + ")"
	}
	return runMutation(requestContext, record, []engines.Statement{{SQL: statement, Arguments: arguments}})
}

func updatePostgres(requestContext context.Context, record *models.Database, schema string, table string, edits []CellEdit) *fiber.Error {
	if len(edits) == 0 || len(edits) > 1000 {
		return shortcuts.ServiceError(http.StatusBadRequest, NothingChanged)
	}
	columns, columnsError := mutationColumns(requestContext, record, schema, table, true)
	if columnsError != nil {
		return columnsError
	}
	grouped := make(map[string][]CellEdit)
	order := make([]string, 0)
	for _, edit := range edits {
		if _, seen := grouped[edit.Key]; !seen {
			order = append(order, edit.Key)
		}
		grouped[edit.Key] = append(grouped[edit.Key], edit)
	}
	statements := make([]engines.Statement, 0, len(order))
	expected := int64(1)
	for _, token := range order {
		identity, identityError := verifyRow(token, record.ID, schema, table, columns)
		if identityError != nil {
			return shortcuts.ServiceError(http.StatusBadRequest, "Invalid or obsolete row identity. Reload the table.")
		}
		arguments := make([]any, 0)
		assignments := make([]string, 0)
		seen := make(map[string]bool)
		for _, edit := range grouped[token] {
			column, exists := columnByName(columns, edit.Column)
			if !exists {
				return shortcuts.ServiceError(http.StatusBadRequest, ColumnMissing)
			}
			if column.ReadOnly || seen[column.Name] {
				return shortcuts.ServiceError(http.StatusBadRequest, "This column cannot be changed.")
			}
			seen[column.Name] = true
			state := edit.State
			if edit.Clear {
				state = "null"
			}
			expression, fieldError := fieldExpression(FieldValue{State: state, Value: edit.Value}, column, &arguments)
			if fieldError != nil {
				return fieldError
			}
			assignments = append(assignments, quoteIdentifier(column.Name)+" = "+expression)
		}
		predicate := identityPredicate(identity, &arguments)
		statements = append(statements, engines.Statement{SQL: "UPDATE " + qualifiedRelation(schema, table) + " SET " + strings.Join(assignments, ", ") + " WHERE " + predicate, Arguments: arguments, ExpectedAffected: &expected})
	}
	return runMutation(requestContext, record, statements)
}

func deletePostgres(requestContext context.Context, record *models.Database, schema string, table string, tokens []string) *fiber.Error {
	if len(tokens) == 0 || len(tokens) > 1000 {
		return shortcuts.ServiceError(http.StatusBadRequest, NothingChosen)
	}
	columns, columnsError := mutationColumns(requestContext, record, schema, table, true)
	if columnsError != nil {
		return columnsError
	}
	statements := make([]engines.Statement, 0, len(tokens))
	seen := make(map[string]bool)
	expected := int64(1)
	for _, token := range tokens {
		if seen[token] {
			continue
		}
		seen[token] = true
		identity, identityError := verifyRow(token, record.ID, schema, table, columns)
		if identityError != nil {
			return shortcuts.ServiceError(http.StatusBadRequest, "Invalid or obsolete row identity. Reload the table.")
		}
		arguments := make([]any, 0)
		predicate := identityPredicate(identity, &arguments)
		statements = append(statements, engines.Statement{SQL: "DELETE FROM " + qualifiedRelation(schema, table) + " WHERE " + predicate, Arguments: arguments, ExpectedAffected: &expected})
	}
	return runMutation(requestContext, record, statements)
}

func runMutation(requestContext context.Context, record *models.Database, statements []engines.Statement) *fiber.Error {
	fresh, permissionError := authorizedDatabase(requestContext, record.Name, permissions.DataWrite)
	if permissionError != nil {
		return permissionError
	}
	if fresh.ID != record.ID {
		return shortcuts.ServiceError(http.StatusConflict, "The database registration changed.")
	}
	_, runError := engines.Run(databaseContext(requestContext, fresh, engines.ReadWrite), fresh.Name, statements...)
	if errors.Is(runError, engines.Conflict) {
		return shortcuts.ServiceError(http.StatusConflict, "A row changed or was removed. Nothing was saved. Reload the table and try again.")
	}
	if runError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, strings.TrimSpace(runError.Error()))
	}
	return nil
}
