package postgres

import (
	"context"
	"strings"

	"sqldash/engines"
	"sqldash/models"
	"sqldash/pgcluster/control"

	"github.com/jackc/pgx/v5"
)

func (self *Backend) Structure(ctx context.Context, record *models.Database, schema, name string) (*engines.Definition, error) {
	connection, connectError := control.ConnectMetadata(ctx, record)
	if connectError != nil {
		return nil, connectError
	}
	defer closeSession(connection)
	definition := &engines.Definition{Relation: engines.Relation{Schema: schema}}
	if name != "" {
		relation, relationError := findRelation(ctx, connection, schema, name)
		if relationError != nil {
			return nil, relationError
		}
		definition.Relation = *relation
		var readError error
		definition.Columns, readError = columns(ctx, connection, schema, name)
		if readError != nil {
			return nil, readError
		}
		definition.Constraints, readError = constraints(ctx, connection, schema, name)
		if readError != nil {
			return nil, readError
		}
		definition.Indexes, readError = indexes(ctx, connection, schema, name)
		if readError != nil {
			return nil, readError
		}
		definition.Triggers, readError = triggers(ctx, connection, schema, name)
		if readError != nil {
			return nil, readError
		}
		definition.SQL, readError = relationDefinition(ctx, connection, definition)
		if readError != nil {
			return nil, readError
		}
	} else {
		var exists bool
		queryError := connection.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_namespace AS namespace WHERE namespace.nspname = $1 AND "+browsableSchemaSelection+")", schema).Scan(&exists)
		if queryError != nil {
			return nil, control.SafeError(queryError)
		}
		if !exists {
			return nil, InvalidRelation
		}
	}
	if schema == "pg_catalog" || schema == "information_schema" {
		return definition, nil
	}
	var readError error
	definition.Functions, readError = functions(ctx, connection, schema)
	if readError != nil {
		return nil, readError
	}
	definition.Sequences, readError = sequences(ctx, connection, schema)
	if readError != nil {
		return nil, readError
	}
	return definition, nil
}

func relationDefinition(ctx context.Context, connection *pgx.Conn, definition *engines.Definition) (string, error) {
	relation := definition.Relation
	identifier := pgx.Identifier{relation.Schema, relation.Name}.Sanitize()
	if relation.Kind == "view" || relation.Kind == "materialized view" {
		var body string
		queryError := connection.QueryRow(ctx, `SELECT pg_catalog.pg_get_viewdef(relation.oid, true)
			FROM pg_catalog.pg_class AS relation JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace
			WHERE namespace.nspname = $1 AND relation.relname = $2`, relation.Schema, relation.Name).Scan(&body)
		if queryError != nil {
			return "", control.SafeError(queryError)
		}
		prefix := "CREATE VIEW "
		if relation.Kind == "materialized view" {
			prefix = "CREATE MATERIALIZED VIEW "
		}
		return prefix + identifier + " AS\n" + strings.TrimSuffix(strings.TrimSpace(body), ";") + ";", nil
	}
	rows, queryError := connection.Query(ctx, columnDefinitionsSelection, relation.Schema, relation.Name)
	if queryError != nil {
		return "", control.SafeError(queryError)
	}
	parts := []string{}
	for rows.Next() {
		var part string
		if scanError := rows.Scan(&part); scanError != nil {
			rows.Close()
			return "", control.SafeError(scanError)
		}
		parts = append(parts, "  "+part)
	}
	queryError = rows.Err()
	rows.Close()
	if queryError != nil {
		return "", control.SafeError(queryError)
	}
	for _, constraint := range definition.Constraints {
		if constraint.Kind != "not null" {
			parts = append(parts, "  CONSTRAINT "+pgx.Identifier{constraint.Name}.Sanitize()+" "+constraint.Definition)
		}
	}
	var persistence, partitionDefinition, partitionBound, parent string
	queryError = connection.QueryRow(ctx, `SELECT relation.relpersistence::text,
		CASE WHEN relation.relkind = 'p' THEN pg_catalog.pg_get_partkeydef(relation.oid) ELSE '' END,
		COALESCE(pg_catalog.pg_get_expr(relation.relpartbound, relation.oid), ''),
		COALESCE((SELECT pg_catalog.format('%I.%I', parent_namespace.nspname, parent_relation.relname)
			FROM pg_catalog.pg_inherits AS inheritance JOIN pg_catalog.pg_class AS parent_relation ON parent_relation.oid = inheritance.inhparent
			JOIN pg_catalog.pg_namespace AS parent_namespace ON parent_namespace.oid = parent_relation.relnamespace
			WHERE inheritance.inhrelid = relation.oid ORDER BY inheritance.inhseqno LIMIT 1), '')
		FROM pg_catalog.pg_class AS relation JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace
		WHERE namespace.nspname = $1 AND relation.relname = $2`, relation.Schema, relation.Name).Scan(&persistence, &partitionDefinition, &partitionBound, &parent)
	if queryError != nil {
		return "", control.SafeError(queryError)
	}
	prefix := "CREATE TABLE "
	if persistence == "u" {
		prefix = "CREATE UNLOGGED TABLE "
	}
	if relation.Kind == "foreign table" {
		return "", engines.Unsupported
	}
	body := prefix + identifier + " (\n" + strings.Join(parts, ",\n") + "\n)"
	if partitionBound != "" {
		body = prefix + identifier + " PARTITION OF " + parent + " " + partitionBound
	} else if parent != "" {
		body += " INHERITS (" + parent + ")"
	}
	if partitionDefinition != "" {
		body += " PARTITION BY " + partitionDefinition
	}
	body += ";"
	for _, index := range definition.Indexes {
		backedByConstraint := false
		for _, constraint := range definition.Constraints {
			if index.Name == constraint.Name && (constraint.Kind == "primary key" || constraint.Kind == "unique" || constraint.Kind == "exclusion") {
				backedByConstraint = true
				break
			}
		}
		if !backedByConstraint {
			body += "\n" + index.Definition + ";"
		}
	}
	for _, trigger := range definition.Triggers {
		body += "\n" + trigger.Definition + ";"
	}
	return body, nil
}

const columnDefinitionsSelection = `SELECT pg_catalog.format('%I %s', attribute.attname, pg_catalog.format_type(attribute.atttypid, attribute.atttypmod)) ||
	CASE WHEN attribute.attcollation <> type_record.typcollation THEN
		pg_catalog.format(' COLLATE %I.%I', collation_namespace.nspname, collation_record.collname) ELSE '' END ||
	CASE WHEN attribute.attidentity <> '' THEN
		CASE attribute.attidentity WHEN 'a' THEN ' GENERATED ALWAYS AS IDENTITY' ELSE ' GENERATED BY DEFAULT AS IDENTITY' END ||
		COALESCE(pg_catalog.format(' (START WITH %s INCREMENT BY %s MINVALUE %s MAXVALUE %s CACHE %s %s)',
			sequence_record.seqstart, sequence_record.seqincrement, sequence_record.seqmin, sequence_record.seqmax,
			sequence_record.seqcache, CASE WHEN sequence_record.seqcycle THEN 'CYCLE' ELSE 'NO CYCLE' END), '')
	WHEN attribute.attgenerated <> '' THEN ' GENERATED ALWAYS AS (' || pg_catalog.pg_get_expr(value.adbin, value.adrelid) || ')' ||
		CASE attribute.attgenerated WHEN 'v' THEN ' VIRTUAL' ELSE ' STORED' END
	WHEN value.adbin IS NOT NULL THEN ' DEFAULT ' || pg_catalog.pg_get_expr(value.adbin, value.adrelid) ELSE '' END ||
	CASE WHEN attribute.attnotnull THEN ' NOT NULL' ELSE '' END
	FROM pg_catalog.pg_class AS relation JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace
	JOIN pg_catalog.pg_attribute AS attribute ON attribute.attrelid = relation.oid AND attribute.attnum > 0 AND NOT attribute.attisdropped
	JOIN pg_catalog.pg_type AS type_record ON type_record.oid = attribute.atttypid
	LEFT JOIN pg_catalog.pg_collation AS collation_record ON collation_record.oid = attribute.attcollation
	LEFT JOIN pg_catalog.pg_namespace AS collation_namespace ON collation_namespace.oid = collation_record.collnamespace
	LEFT JOIN pg_catalog.pg_attrdef AS value ON value.adrelid = relation.oid AND value.adnum = attribute.attnum
	LEFT JOIN LATERAL (
		SELECT sequence_record.* FROM pg_catalog.pg_sequence AS sequence_record
		JOIN pg_catalog.pg_depend AS dependency ON dependency.objid = sequence_record.seqrelid
		WHERE attribute.attidentity <> '' AND dependency.classid = 'pg_catalog.pg_class'::regclass
		AND dependency.refclassid = 'pg_catalog.pg_class'::regclass AND dependency.refobjid = relation.oid
		AND dependency.refobjsubid = attribute.attnum AND dependency.deptype = 'i'
	) AS sequence_record ON true
	WHERE namespace.nspname = $1 AND relation.relname = $2 ORDER BY attribute.attnum`
