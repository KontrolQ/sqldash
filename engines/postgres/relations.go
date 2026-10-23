package postgres

import (
	"context"
	"errors"

	"sqldash/engines"
	"sqldash/models"
	"sqldash/pgcluster/control"

	"github.com/jackc/pgx/v5"
)

const browsableSchemaSelection = "(namespace.nspname = 'pg_catalog' OR namespace.nspname !~ '^pg_') AND namespace.nspname NOT IN ('" + control.ControlSchema + "', '" + control.ExtensionsSchema + "')"

const relationFields = `namespace.nspname, relation.relname,
	CASE relation.relkind WHEN 'r' THEN 'table' WHEN 'p' THEN 'partitioned table'
	WHEN 'v' THEN 'view' WHEN 'm' THEN 'materialized view' WHEN 'f' THEN 'foreign table' END`

const relationSource = ` FROM pg_catalog.pg_class AS relation
	JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace
	WHERE relation.relkind IN ('r', 'p', 'v', 'm', 'f') AND ` + browsableSchemaSelection

const relationSelection = "SELECT " + relationFields + ", relation.reltuples::bigint" + relationSource

func (self *Backend) Schemas(ctx context.Context, record *models.Database) ([]string, error) {
	connection, connectError := control.ConnectMetadata(ctx, record)
	if connectError != nil {
		return nil, connectError
	}
	defer closeSession(connection)
	rows, queryError := connection.Query(ctx, "SELECT namespace.nspname FROM pg_catalog.pg_namespace AS namespace WHERE "+browsableSchemaSelection+" ORDER BY namespace.nspname")
	if queryError != nil {
		return nil, control.SafeError(queryError)
	}
	defer rows.Close()
	schemas := []string{}
	for rows.Next() {
		var schema string
		if scanError := rows.Scan(&schema); scanError != nil {
			return nil, control.SafeError(scanError)
		}
		schemas = append(schemas, schema)
	}
	return schemas, control.SafeError(rows.Err())
}

func (self *Backend) Relations(ctx context.Context, record *models.Database, schema string) ([]engines.Relation, error) {
	connection, connectError := control.ConnectMetadata(ctx, record)
	if connectError != nil {
		return nil, connectError
	}
	defer closeSession(connection)
	rows, queryError := connection.Query(ctx, relationSelection+` AND ($1 = '' OR namespace.nspname = $1) ORDER BY namespace.nspname, relation.relname`, schema)
	if queryError != nil {
		return nil, control.SafeError(queryError)
	}
	defer rows.Close()
	relations := []engines.Relation{}
	for rows.Next() {
		var relation engines.Relation
		if scanError := rows.Scan(&relation.Schema, &relation.Name, &relation.Kind, &relation.Rows); scanError != nil {
			return nil, control.SafeError(scanError)
		}
		relations = append(relations, relation)
	}
	return relations, control.SafeError(rows.Err())
}

func (self *Backend) RelationColumns(ctx context.Context, record *models.Database) ([]engines.RelationColumns, error) {
	connection, connectError := control.ConnectMetadata(ctx, record)
	if connectError != nil {
		return nil, connectError
	}
	defer closeSession(connection)
	rows, queryError := connection.Query(ctx, "SELECT "+relationFields+`,
		ARRAY(SELECT attribute.attname::text FROM pg_catalog.pg_attribute AS attribute
			WHERE attribute.attrelid = relation.oid AND attribute.attnum > 0 AND NOT attribute.attisdropped
			ORDER BY attribute.attnum)`+relationSource+" ORDER BY namespace.nspname, relation.relname")
	if queryError != nil {
		return nil, control.SafeError(queryError)
	}
	defer rows.Close()
	relations := []engines.RelationColumns{}
	for rows.Next() {
		var relation engines.RelationColumns
		if scanError := rows.Scan(&relation.Schema, &relation.Name, &relation.Kind, &relation.Columns); scanError != nil {
			return nil, control.SafeError(scanError)
		}
		relations = append(relations, relation)
	}
	return relations, control.SafeError(rows.Err())
}

func findRelation(ctx context.Context, connection *pgx.Conn, schema, name string) (*engines.Relation, error) {
	var relation engines.Relation
	queryError := connection.QueryRow(ctx, relationSelection+" AND namespace.nspname = $1 AND relation.relname = $2", schema, name).Scan(&relation.Schema, &relation.Name, &relation.Kind, &relation.Rows)
	if errors.Is(queryError, pgx.ErrNoRows) {
		return nil, InvalidRelation
	}
	if queryError != nil {
		return nil, control.SafeError(queryError)
	}
	return &relation, nil
}
