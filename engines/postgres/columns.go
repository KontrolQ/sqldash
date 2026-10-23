package postgres

import (
	"context"

	"sqldash/engines"
	"sqldash/models"
	"sqldash/pgcluster/control"

	"github.com/jackc/pgx/v5"
)

const columnsSelection = `SELECT attribute.attname, pg_catalog.format_type(attribute.atttypid, attribute.atttypmod),
	attribute.attnotnull, COALESCE(primary_key.position, 0)::integer,
	COALESCE(pg_catalog.pg_get_expr(value.adbin, value.adrelid), ''), attribute.attgenerated <> '', attribute.attidentity <> '',
	COALESCE(reference.relname, ''), COALESCE(reference.schema_name, ''), COALESCE(reference.columns, ARRAY[]::text[])
	FROM pg_catalog.pg_class AS relation JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace
	JOIN pg_catalog.pg_attribute AS attribute ON attribute.attrelid = relation.oid AND attribute.attnum > 0 AND NOT attribute.attisdropped
	LEFT JOIN pg_catalog.pg_attrdef AS value ON value.adrelid = relation.oid AND value.adnum = attribute.attnum
	LEFT JOIN LATERAL (
		SELECT key.position FROM pg_catalog.pg_constraint AS constraint_record,
		unnest(constraint_record.conkey) WITH ORDINALITY AS key(number, position)
		WHERE constraint_record.conrelid = relation.oid AND constraint_record.contype = 'p' AND key.number = attribute.attnum
	) AS primary_key ON true
	LEFT JOIN LATERAL (
		SELECT target.relname, target_namespace.nspname AS schema_name,
		ARRAY(SELECT target_attribute.attname::text FROM unnest(foreign_key.confkey) WITH ORDINALITY AS key(number, position)
			JOIN pg_catalog.pg_attribute AS target_attribute ON target_attribute.attrelid = foreign_key.confrelid AND target_attribute.attnum = key.number ORDER BY key.position) AS columns
		FROM pg_catalog.pg_constraint AS foreign_key
		JOIN pg_catalog.pg_class AS target ON target.oid = foreign_key.confrelid
		JOIN pg_catalog.pg_namespace AS target_namespace ON target_namespace.oid = target.relnamespace
		WHERE foreign_key.conrelid = relation.oid AND foreign_key.contype = 'f' AND attribute.attnum = ANY(foreign_key.conkey)
		ORDER BY foreign_key.oid LIMIT 1
	) AS reference ON true
	WHERE namespace.nspname = $1 AND relation.relname = $2 ORDER BY attribute.attnum`

func (self *Backend) Columns(ctx context.Context, record *models.Database, schema, name string) ([]engines.Column, error) {
	connection, connectError := control.ConnectMetadata(ctx, record)
	if connectError != nil {
		return nil, connectError
	}
	defer closeSession(connection)
	if _, relationError := findRelation(ctx, connection, schema, name); relationError != nil {
		return nil, relationError
	}
	return columns(ctx, connection, schema, name)
}

func columns(ctx context.Context, connection *pgx.Conn, schema, name string) ([]engines.Column, error) {
	rows, queryError := connection.Query(ctx, columnsSelection, schema, name)
	if queryError != nil {
		return nil, control.SafeError(queryError)
	}
	defer rows.Close()
	columns := []engines.Column{}
	for rows.Next() {
		var column engines.Column
		if scanError := rows.Scan(&column.Name, &column.Kind, &column.NotNull, &column.PrimaryPosition, &column.Default,
			&column.Generated, &column.Identity, &column.References, &column.ReferenceSchema, &column.ReferenceColumns); scanError != nil {
			return nil, control.SafeError(scanError)
		}
		column.PrimaryKey = column.PrimaryPosition > 0
		columns = append(columns, column)
	}
	return columns, control.SafeError(rows.Err())
}
