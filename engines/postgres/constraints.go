package postgres

import (
	"context"

	"sqldash/engines"
	"sqldash/pgcluster/control"

	"github.com/jackc/pgx/v5"
)

func constraints(ctx context.Context, connection *pgx.Conn, schema, name string) ([]engines.Constraint, error) {
	rows, queryError := connection.Query(ctx, `SELECT constraint_record.conname,
		CASE constraint_record.contype WHEN 'p' THEN 'primary key' WHEN 'f' THEN 'foreign key' WHEN 'u' THEN 'unique'
		WHEN 'c' THEN 'check' WHEN 'x' THEN 'exclusion' WHEN 'n' THEN 'not null' ELSE constraint_record.contype::text END,
		ARRAY(SELECT attribute.attname::text FROM unnest(constraint_record.conkey) WITH ORDINALITY AS key(number, position)
			JOIN pg_catalog.pg_attribute AS attribute ON attribute.attrelid = relation.oid AND attribute.attnum = key.number ORDER BY key.position),
		COALESCE(target_namespace.nspname, ''), COALESCE(target.relname, ''),
		ARRAY(SELECT attribute.attname::text FROM unnest(constraint_record.confkey) WITH ORDINALITY AS key(number, position)
			JOIN pg_catalog.pg_attribute AS attribute ON attribute.attrelid = target.oid AND attribute.attnum = key.number ORDER BY key.position),
		pg_catalog.pg_get_constraintdef(constraint_record.oid, true)
		FROM pg_catalog.pg_constraint AS constraint_record JOIN pg_catalog.pg_class AS relation ON relation.oid = constraint_record.conrelid
		JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace
		LEFT JOIN pg_catalog.pg_class AS target ON target.oid = constraint_record.confrelid
		LEFT JOIN pg_catalog.pg_namespace AS target_namespace ON target_namespace.oid = target.relnamespace
		WHERE namespace.nspname = $1 AND relation.relname = $2 ORDER BY constraint_record.conname`, schema, name)
	if queryError != nil {
		return nil, control.SafeError(queryError)
	}
	defer rows.Close()
	constraints := []engines.Constraint{}
	for rows.Next() {
		var constraint engines.Constraint
		if scanError := rows.Scan(&constraint.Name, &constraint.Kind, &constraint.Columns, &constraint.ReferenceSchema,
			&constraint.ReferenceTable, &constraint.ReferenceColumns, &constraint.Definition); scanError != nil {
			return nil, control.SafeError(scanError)
		}
		constraints = append(constraints, constraint)
	}
	return constraints, control.SafeError(rows.Err())
}

func indexes(ctx context.Context, connection *pgx.Conn, schema, name string) ([]engines.Index, error) {
	rows, queryError := connection.Query(ctx, `SELECT index_relation.relname, index_record.indisunique, index_record.indisprimary,
		ARRAY(SELECT pg_catalog.pg_get_indexdef(index_record.indexrelid, position, true)
			FROM generate_series(1, index_record.indnatts) AS position ORDER BY position),
		pg_catalog.pg_get_indexdef(index_record.indexrelid)
		FROM pg_catalog.pg_index AS index_record JOIN pg_catalog.pg_class AS relation ON relation.oid = index_record.indrelid
		JOIN pg_catalog.pg_class AS index_relation ON index_relation.oid = index_record.indexrelid
		JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace
		WHERE namespace.nspname = $1 AND relation.relname = $2 ORDER BY index_relation.relname`, schema, name)
	if queryError != nil {
		return nil, control.SafeError(queryError)
	}
	defer rows.Close()
	indexes := []engines.Index{}
	for rows.Next() {
		var index engines.Index
		if scanError := rows.Scan(&index.Name, &index.Unique, &index.Primary, &index.Columns, &index.Definition); scanError != nil {
			return nil, control.SafeError(scanError)
		}
		indexes = append(indexes, index)
	}
	return indexes, control.SafeError(rows.Err())
}

func triggers(ctx context.Context, connection *pgx.Conn, schema, name string) ([]engines.NamedDefinition, error) {
	rows, queryError := connection.Query(ctx, `SELECT trigger_record.tgname, pg_catalog.pg_get_triggerdef(trigger_record.oid, true)
		FROM pg_catalog.pg_trigger AS trigger_record JOIN pg_catalog.pg_class AS relation ON relation.oid = trigger_record.tgrelid
		JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace
		WHERE namespace.nspname = $1 AND relation.relname = $2 AND NOT trigger_record.tgisinternal ORDER BY trigger_record.tgname`, schema, name)
	if queryError != nil {
		return nil, control.SafeError(queryError)
	}
	return namedDefinitions(rows)
}

func namedDefinitions(rows pgx.Rows) ([]engines.NamedDefinition, error) {
	defer rows.Close()
	definitions := []engines.NamedDefinition{}
	for rows.Next() {
		var definition engines.NamedDefinition
		if scanError := rows.Scan(&definition.Name, &definition.Definition); scanError != nil {
			return nil, control.SafeError(scanError)
		}
		definitions = append(definitions, definition)
	}
	return definitions, control.SafeError(rows.Err())
}
