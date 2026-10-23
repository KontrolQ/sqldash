package explorer

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"sqldash/engines"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/services/access"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

type ObjectRequest struct {
	Action              string   `form:"action"`
	Kind                string   `form:"kind"`
	Schema              string   `form:"schema"`
	TargetSchema        string   `form:"target_schema"`
	Table               string   `form:"table"`
	Name                string   `form:"name"`
	NewName             string   `form:"new_name"`
	Column              string   `form:"column"`
	Type                string   `form:"type"`
	Value               string   `form:"value"`
	Expression          string   `form:"expression"`
	Columns             []string `form:"columns"`
	ColumnNames         []string `form:"column_name"`
	ColumnTypes         []string `form:"column_type"`
	ColumnModes         []string `form:"column_mode"`
	ColumnRequired      []string `form:"column_required"`
	ColumnDefaultStates []string `form:"column_default_state"`
	ColumnDefaultValues []string `form:"column_default_value"`
	ColumnExpressions   []string `form:"column_expression"`
	Mode                string   `form:"mode"`
	DefaultState        string   `form:"default_state"`
	ReferenceSchema     string   `form:"reference_schema"`
	ReferenceTable      string   `form:"reference_table"`
	ReferenceColumns    []string `form:"reference_columns"`
	ConstraintKind      string   `form:"constraint_kind"`
	Unique              bool     `form:"unique"`
	NotNull             bool     `form:"not_null"`
	Cascade             bool     `form:"cascade"`
	Confirmed           bool     `form:"confirmed"`
	Language            string   `form:"language"`
	Signature           string   `form:"signature"`
	ArgumentTypes       []string `form:"argument_types"`
	Returns             string   `form:"returns"`
	Start               string   `form:"start"`
	Increment           string   `form:"increment"`
	Minimum             string   `form:"minimum"`
	Maximum             string   `form:"maximum"`
}

func ManageObject(requestContext context.Context, databaseName string, asked ObjectRequest) *fiber.Error {
	record, permissionError := authorizedDatabase(requestContext, databaseName, permissions.DataRead)
	if permissionError != nil {
		return permissionError
	}
	actor := access.FromContext(requestContext)
	fresh, loadError := access.Load(actor.Account.ID)
	if loadError != nil || record.Kind() != models.EnginePostgres || !fresh.Can(permissions.PostgresObjectsManage) {
		return shortcuts.ServiceError(http.StatusForbidden, "Database structure access denied.")
	}
	statement, statementError := objectStatement(requestContext, record, asked)
	if statementError != nil {
		return statementError
	}
	fresh, loadError = access.Load(actor.Account.ID)
	if loadError != nil || !fresh.Can(permissions.PostgresObjectsManage) {
		return shortcuts.ServiceError(http.StatusForbidden, "Database structure access denied.")
	}
	_, runError := engines.Run(databaseContext(requestContext, record, engines.Objects), record.Name, engines.Statement{SQL: statement})
	if runError != nil {
		return shortcuts.ServiceError(http.StatusBadRequest, strings.TrimSpace(runError.Error()))
	}
	return nil
}

func postgresTypeChoices() []ChoiceView {
	names := []string{"text", "boolean", "smallint", "integer", "bigint", "numeric", "real", "double precision", "uuid", "json", "jsonb", "bytea", "date", "time", "time with time zone", "timestamp", "timestamp with time zone", "interval", "text[]", "integer[]", "bigint[]", "uuid[]", "jsonb[]"}
	choices := make([]ChoiceView, 0, len(names))
	for _, name := range names {
		choices = append(choices, ChoiceView{Value: name, Label: name})
	}
	return choices
}

func allowedObjectType(kind string) bool {
	for _, choice := range postgresTypeChoices() {
		if choice.Value == kind {
			return true
		}
	}
	return false
}

func (self ObjectRequest) SelectedSchema() string {
	if self.Action == "create" && self.Kind == "table" && self.TargetSchema != "" {
		return self.TargetSchema
	}
	return self.Schema
}

func managedSchema(name string) bool {
	return safeObjectName(name) && mutablePostgresSchema(name)
}

func objectStatement(requestContext context.Context, record *models.Database, asked ObjectRequest) (string, *fiber.Error) {
	asked.Schema = asked.SelectedSchema()
	if !managedSchema(asked.Schema) {
		return "", consoleRequestError("Choose a user schema.")
	}
	catalog, catalogError := catalogFor(record)
	if catalogError != nil {
		return "", catalogError
	}
	if asked.Action == "create" && asked.Kind == "schema" {
		return "CREATE SCHEMA " + quoteIdentifier(asked.Schema), nil
	}
	relations, relationsError := postgresRelations(requestContext, record, "")
	if relationsError != nil {
		return "", relationsError
	}
	schemas, schemaError := postgresSchemas(requestContext, record, relations)
	if schemaError != nil {
		return "", schemaError
	}
	foundSchema := false
	for _, schema := range schemas {
		if schema.Value == asked.Schema {
			foundSchema = true
		}
	}
	if !foundSchema {
		return "", consoleRequestError("The schema no longer exists.")
	}
	if asked.Action == "create" || (asked.Action == "replace" && asked.Kind == "function") {
		return createObjectStatement(requestContext, record, asked)
	}
	if asked.Action == "drop" && !asked.Confirmed {
		return "", consoleRequestError("Confirm this destructive change.")
	}
	if asked.Kind == "schema" {
		switch asked.Action {
		case "rename":
			if safeObjectName(asked.NewName) && managedSchema(asked.NewName) {
				return "ALTER SCHEMA " + quoteIdentifier(asked.Schema) + " RENAME TO " + quoteIdentifier(asked.NewName), nil
			}
		case "drop":
			return "DROP SCHEMA " + quoteIdentifier(asked.Schema) + cascadeClause(asked.Cascade), nil
		}
		return "", consoleRequestError("Unknown schema operation.")
	}
	if asked.Kind == "function" || asked.Kind == "sequence" {
		definition, readError := catalog.Structure(databaseContext(requestContext, record, engines.ReadOnly), record, asked.Schema, "")
		if readError != nil || definition == nil {
			return "", consoleRequestError("The object catalog is unavailable.")
		}
		return namedObjectStatement(definition, asked)
	}
	definition, definitionError := postgresDefinition(requestContext, record, asked.Schema, asked.Table)
	if definitionError != nil {
		return "", definitionError
	}
	qualified := qualifiedRelation(asked.Schema, asked.Table)
	if asked.Kind == "table" || asked.Kind == "view" || asked.Kind == "materialized view" {
		if definition.Relation.Kind != asked.Kind && !(asked.Kind == "table" && definition.Relation.Kind == "partitioned table") {
			return "", consoleRequestError("The object kind does not match.")
		}
		switch asked.Action {
		case "rename":
			if !safeObjectName(asked.NewName) {
				return "", consoleRequestError("Enter a valid new name.")
			}
			return "ALTER " + strings.ToUpper(asked.Kind) + " " + qualified + " RENAME TO " + quoteIdentifier(asked.NewName), nil
		case "drop":
			return "DROP " + strings.ToUpper(asked.Kind) + " " + qualified + cascadeClause(asked.Cascade), nil
		case "refresh":
			if asked.Kind == "materialized view" {
				return "REFRESH MATERIALIZED VIEW " + qualified, nil
			}
		case "replace":
			if asked.Kind == "view" && selectFragment(asked.Expression) {
				return "CREATE OR REPLACE VIEW " + qualified + " AS " + asked.Expression, nil
			}
		}
		return "", consoleRequestError("Unknown relation operation.")
	}
	if !writableRelation(definition.Relation.Kind) {
		return "", consoleRequestError("Choose a table for this operation.")
	}
	columns := postgresColumnViews(definition.Columns)
	if asked.Kind == "column" {
		return columnObjectStatement(qualified, columns, asked)
	}
	if asked.Kind == "constraint" {
		if asked.Action == "add" {
			return addConstraintStatement(requestContext, record, qualified, columns, asked)
		}
		found := false
		for _, constraint := range definition.Constraints {
			if constraint.Name == asked.Name {
				found = true
			}
		}
		if !found {
			return "", consoleRequestError("The constraint no longer exists.")
		}
		if asked.Action == "drop" {
			return "ALTER TABLE " + qualified + " DROP CONSTRAINT " + quoteIdentifier(asked.Name) + cascadeClause(asked.Cascade), nil
		}
		if asked.Action == "rename" && safeObjectName(asked.NewName) {
			return "ALTER TABLE " + qualified + " RENAME CONSTRAINT " + quoteIdentifier(asked.Name) + " TO " + quoteIdentifier(asked.NewName), nil
		}
	}
	if asked.Kind == "index" {
		for _, index := range definition.Indexes {
			if index.Name != asked.Name {
				continue
			}
			if index.Primary {
				return "", consoleRequestError("Manage this index through its primary key constraint.")
			}
			if asked.Action == "drop" {
				return "DROP INDEX " + qualifiedRelation(asked.Schema, asked.Name) + cascadeClause(asked.Cascade), nil
			}
			if asked.Action == "rename" && safeObjectName(asked.NewName) {
				return "ALTER INDEX " + qualifiedRelation(asked.Schema, asked.Name) + " RENAME TO " + quoteIdentifier(asked.NewName), nil
			}
		}
	}
	return "", consoleRequestError("Unknown object operation.")
}

func createObjectStatement(requestContext context.Context, record *models.Database, asked ObjectRequest) (string, *fiber.Error) {
	if !safeObjectName(asked.Name) {
		return "", consoleRequestError("Enter a valid object name.")
	}
	qualified := qualifiedRelation(asked.Schema, asked.Name)
	switch asked.Kind {
	case "table":
		if len(asked.ColumnNames) == 0 || len(asked.ColumnNames) != len(asked.ColumnTypes) || len(asked.ColumnNames) > 128 {
			return "", consoleRequestError("Provide names and types for every column.")
		}
		columns := make([]string, 0)
		seen := make(map[string]bool)
		for index, name := range asked.ColumnNames {
			if !safeObjectName(name) || seen[name] || !allowedObjectType(asked.ColumnTypes[index]) {
				return "", consoleRequestError("Invalid column name or type.")
			}
			seen[name] = true
			declaration, declarationError := columnDeclaration(name, asked.ColumnTypes[index], atIndex(asked.ColumnModes, index), atIndex(asked.ColumnRequired, index) == "required", atIndex(asked.ColumnDefaultStates, index), atIndex(asked.ColumnDefaultValues, index), atIndex(asked.ColumnExpressions, index))
			if declarationError != nil {
				return "", declarationError
			}
			columns = append(columns, declaration)
		}
		return "CREATE TABLE " + qualified + " (" + strings.Join(columns, ", ") + ")", nil
	case "view", "materialized view":
		if !selectFragment(asked.Expression) {
			return "", consoleRequestError("Provide one SELECT or WITH query.")
		}
		return "CREATE " + strings.ToUpper(asked.Kind) + " " + qualified + " AS " + asked.Expression, nil
	case "index":
		definition, definitionError := postgresDefinition(requestContext, record, asked.Schema, asked.Table)
		if definitionError != nil {
			return "", definitionError
		}
		columns, columnsError := selectedObjectColumns(postgresColumnViews(definition.Columns), asked.Columns)
		if columnsError != nil {
			return "", columnsError
		}
		unique := ""
		if asked.Unique {
			unique = "UNIQUE "
		}
		return "CREATE " + unique + "INDEX " + quoteIdentifier(asked.Name) + " ON " + qualifiedRelation(asked.Schema, asked.Table) + " (" + strings.Join(columns, ", ") + ")", nil
	case "sequence":
		options, optionsError := sequenceOptions(asked)
		if optionsError != nil {
			return "", optionsError
		}
		return "CREATE SEQUENCE " + qualified + options, nil
	case "function":
		if !allowedObjectType(asked.Returns) && asked.Returns != "void" {
			return "", consoleRequestError("Choose a supported return type.")
		}
		if asked.Language != "sql" && asked.Language != "plpgsql" {
			return "", consoleRequestError("Choose SQL or PL/pgSQL.")
		}
		arguments := make([]string, 0, len(asked.ArgumentTypes))
		for index, kind := range asked.ArgumentTypes {
			if kind == "" {
				continue
			}
			if !allowedObjectType(kind) {
				return "", consoleRequestError("Choose supported argument types.")
			}
			arguments = append(arguments, quoteIdentifier("argument"+strconv.Itoa(index+1))+" "+kind)
		}
		if strings.TrimSpace(asked.Expression) == "" || strings.ContainsRune(asked.Expression, 0) {
			return "", consoleRequestError("Provide a function body.")
		}
		prefix := "CREATE FUNCTION "
		if asked.Action == "replace" {
			prefix = "CREATE OR REPLACE FUNCTION "
		}
		return prefix + qualified + " (" + strings.Join(arguments, ", ") + ") RETURNS " + asked.Returns + " LANGUAGE " + asked.Language + " SECURITY INVOKER SET search_path = " + quoteIdentifier(asked.Schema) + ", pg_catalog AS " + quoteLiteral(asked.Expression), nil
	}
	return "", consoleRequestError("Unknown object kind.")
}

func columnObjectStatement(table string, columns []ColumnView, asked ObjectRequest) (string, *fiber.Error) {
	if asked.Action == "add" {
		if !safeObjectName(asked.Column) || columnNamed(columns, asked.Column) || !allowedObjectType(asked.Type) {
			return "", consoleRequestError("Invalid column name or type.")
		}
		declaration, declarationError := columnDeclaration(asked.Column, asked.Type, asked.Mode, asked.NotNull, asked.DefaultState, asked.Value, asked.Expression)
		if declarationError != nil {
			return "", declarationError
		}
		return "ALTER TABLE " + table + " ADD COLUMN " + declaration, nil
	}
	column, found := columnByName(columns, asked.Column)
	if !found {
		return "", consoleRequestError("The column no longer exists.")
	}
	prefix := "ALTER TABLE " + table + " "
	if asked.Action == "drop" {
		return prefix + "DROP COLUMN " + quoteIdentifier(column.Name) + cascadeClause(asked.Cascade), nil
	}
	if asked.Action == "rename" && safeObjectName(asked.NewName) {
		return prefix + "RENAME COLUMN " + quoteIdentifier(column.Name) + " TO " + quoteIdentifier(asked.NewName), nil
	}
	prefix += "ALTER COLUMN " + quoteIdentifier(column.Name) + " "
	if column.ReadOnly {
		return "", consoleRequestError("Generated and identity columns are read-only.")
	}
	switch asked.Action {
	case "type":
		if allowedObjectType(asked.Type) {
			return prefix + "TYPE " + asked.Type + " USING " + quoteIdentifier(column.Name) + "::" + asked.Type, nil
		}
	case "default":
		return prefix + "SET DEFAULT " + quoteLiteral(asked.Value) + "::" + column.Kind, nil
	case "drop_default":
		return prefix + "DROP DEFAULT", nil
	case "not_null":
		return prefix + "SET NOT NULL", nil
	case "nullable":
		return prefix + "DROP NOT NULL", nil
	}
	return "", consoleRequestError("Unknown column operation.")
}

func atIndex(values []string, index int) string {
	if index < len(values) {
		return values[index]
	}
	return ""
}

func columnDeclaration(name string, kind string, mode string, required bool, defaultState string, defaultValue string, expression string) (string, *fiber.Error) {
	if !safeObjectName(name) || !allowedObjectType(kind) {
		return "", consoleRequestError("Invalid column name or type.")
	}
	statement := quoteIdentifier(name) + " " + kind
	switch mode {
	case "identity":
		if kind != "smallint" && kind != "integer" && kind != "bigint" {
			return "", consoleRequestError("Identity columns need an integer type.")
		}
		statement += " GENERATED ALWAYS AS IDENTITY"
	case "generated":
		if !sqlFragment(expression) {
			return "", consoleRequestError("Provide one generated-column expression.")
		}
		statement += " GENERATED ALWAYS AS (" + expression + ") STORED"
	case "", "plain":
	default:
		return "", consoleRequestError("Choose a column generation mode.")
	}
	if required {
		statement += " NOT NULL"
	}
	if defaultState == "" || defaultState == "unset" {
		return statement, nil
	}
	if mode == "identity" || mode == "generated" {
		return "", consoleRequestError("Generated columns cannot also have a default.")
	}
	switch defaultState {
	case "value":
		statement += " DEFAULT " + quoteLiteral(defaultValue) + "::" + kind
	case "null":
		if required {
			return "", consoleRequestError("A required column cannot default to NULL.")
		}
		statement += " DEFAULT NULL"
	default:
		return "", consoleRequestError("Choose a default value state.")
	}
	return statement, nil
}

func selectedObjectColumns(columns []ColumnView, wanted []string) ([]string, *fiber.Error) {
	if len(wanted) == 0 {
		return nil, consoleRequestError("Choose at least one column.")
	}
	selected := make([]string, 0, len(wanted))
	seen := make(map[string]bool)
	for _, name := range wanted {
		if !columnNamed(columns, name) || seen[name] {
			return nil, consoleRequestError("Choose existing distinct columns.")
		}
		seen[name] = true
		selected = append(selected, quoteIdentifier(name))
	}
	return selected, nil
}

func addConstraintStatement(requestContext context.Context, record *models.Database, table string, columns []ColumnView, asked ObjectRequest) (string, *fiber.Error) {
	if !safeObjectName(asked.Name) {
		return "", consoleRequestError("Provide a constraint name.")
	}
	prefix := "ALTER TABLE " + table + " ADD CONSTRAINT " + quoteIdentifier(asked.Name) + " "
	if asked.ConstraintKind == "check" {
		if !sqlFragment(asked.Expression) {
			return "", consoleRequestError("Provide one check expression.")
		}
		return prefix + "CHECK (" + asked.Expression + ")", nil
	}
	selected, selectedError := selectedObjectColumns(columns, asked.Columns)
	if selectedError != nil {
		return "", selectedError
	}
	list := "(" + strings.Join(selected, ", ") + ")"
	switch asked.ConstraintKind {
	case "primary":
		return prefix + "PRIMARY KEY " + list, nil
	case "unique":
		return prefix + "UNIQUE " + list, nil
	case "foreign":
		if !managedSchema(asked.ReferenceSchema) || len(asked.Columns) != len(asked.ReferenceColumns) {
			return "", consoleRequestError("Provide every referenced column in key order.")
		}
		referenced, referenceError := postgresColumns(requestContext, record, asked.ReferenceSchema, asked.ReferenceTable)
		if referenceError != nil {
			return "", referenceError
		}
		targets, targetError := selectedObjectColumns(referenced, asked.ReferenceColumns)
		if targetError != nil {
			return "", targetError
		}
		return prefix + "FOREIGN KEY " + list + " REFERENCES " + qualifiedRelation(asked.ReferenceSchema, asked.ReferenceTable) + " (" + strings.Join(targets, ", ") + ")", nil
	}
	return "", consoleRequestError("Choose a constraint kind.")
}

func namedObjectStatement(definition *engines.Definition, asked ObjectRequest) (string, *fiber.Error) {
	if asked.Kind == "sequence" {
		for _, sequence := range definition.Sequences {
			if !matchesQualifiedName(sequence.Name, asked.Schema, asked.Name) {
				continue
			}
			qualified := sequence.Name
			switch asked.Action {
			case "drop":
				return "DROP SEQUENCE " + qualified + cascadeClause(asked.Cascade), nil
			case "rename":
				if safeObjectName(asked.NewName) {
					return "ALTER SEQUENCE " + qualified + " RENAME TO " + quoteIdentifier(asked.NewName), nil
				}
			case "alter":
				options, optionsError := sequenceOptions(asked)
				if optionsError != nil {
					return "", optionsError
				}
				if options == "" {
					return "", consoleRequestError("Provide sequence options.")
				}
				return "ALTER SEQUENCE " + qualified + options, nil
			}
		}
	} else {
		for _, function := range definition.Functions {
			if function.Name != asked.Signature {
				continue
			}
			if asked.Action == "drop" {
				return "DROP FUNCTION " + function.Name + cascadeClause(asked.Cascade), nil
			}
			if asked.Action == "rename" && safeObjectName(asked.NewName) {
				return "ALTER FUNCTION " + function.Name + " RENAME TO " + quoteIdentifier(asked.NewName), nil
			}
			if asked.Action == "replace" {
				return "", consoleRequestError("Create a replacement through the function form with the same name and argument types.")
			}
		}
	}
	return "", consoleRequestError("The object no longer exists or the operation is unsupported.")
}

func sequenceOptions(asked ObjectRequest) (string, *fiber.Error) {
	options := ""
	for _, option := range []struct {
		Name  string
		Value string
	}{{" START WITH ", asked.Start}, {" INCREMENT BY ", asked.Increment}, {" MINVALUE ", asked.Minimum}, {" MAXVALUE ", asked.Maximum}} {
		if option.Value == "" {
			continue
		}
		value, parseError := strconv.ParseInt(option.Value, 10, 64)
		if parseError != nil || (option.Name == " INCREMENT BY " && value == 0) {
			return "", consoleRequestError("Sequence options must be valid 64-bit integers.")
		}
		options += option.Name + strconv.FormatInt(value, 10)
	}
	return options, nil
}

func cascadeClause(cascade bool) string {
	if cascade {
		return " CASCADE"
	}
	return " RESTRICT"
}

func quoteLiteral(value string) string {
	return "E'" + strings.ReplaceAll(strings.ReplaceAll(value, "\\", "\\\\"), "'", "''") + "'"
}

func selectFragment(value string) bool {
	text := strings.ToUpper(strings.TrimSpace(value))
	return (strings.HasPrefix(text, "SELECT ") || strings.HasPrefix(text, "SELECT\n") || strings.HasPrefix(text, "WITH ") || strings.HasPrefix(text, "WITH\n")) && sqlFragment(value)
}

func sqlFragment(value string) bool {
	if strings.TrimSpace(value) == "" || strings.ContainsRune(value, 0) {
		return false
	}
	depth := 0
	for index := 0; index < len(value); index++ {
		switch value[index] {
		case ';':
			return false
		case '-':
			if index+1 < len(value) && value[index+1] == '-' {
				return false
			}
		case '/':
			if index+1 < len(value) && value[index+1] == '*' {
				return false
			}
		case '$':
			return false
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return false
			}
		case '\'', '"':
			quote := value[index]
			closed := false
			for index++; index < len(value); index++ {
				if value[index] == '\\' {
					return false
				}
				if value[index] != quote {
					continue
				}
				if index+1 < len(value) && value[index+1] == quote {
					index++
					continue
				}
				closed = true
				break
			}
			if !closed {
				return false
			}
		}
	}
	return depth == 0
}
