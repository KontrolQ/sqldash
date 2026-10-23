package postgres

import (
	"context"

	"sqldash/engines"
	"sqldash/pgcluster/control"

	"github.com/jackc/pgx/v5"
)

func functions(ctx context.Context, connection *pgx.Conn, schema string) ([]engines.NamedDefinition, error) {
	rows, queryError := connection.Query(ctx, `SELECT pg_catalog.format('%I.%I(%s)', namespace.nspname, routine.proname,
		pg_catalog.pg_get_function_identity_arguments(routine.oid)), pg_catalog.pg_get_functiondef(routine.oid)
		FROM pg_catalog.pg_proc AS routine JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = routine.pronamespace
		WHERE namespace.nspname = $1 AND routine.prokind IN ('f', 'p', 'w')
		ORDER BY routine.proname, routine.oid`, schema)
	if queryError != nil {
		return nil, control.SafeError(queryError)
	}
	return namedDefinitions(rows)
}

func sequences(ctx context.Context, connection *pgx.Conn, schema string) ([]engines.NamedDefinition, error) {
	rows, queryError := connection.Query(ctx, `SELECT pg_catalog.format('%I.%I', namespace.nspname, relation.relname),
		pg_catalog.format('CREATE SEQUENCE %I.%I AS %s INCREMENT BY %s MINVALUE %s MAXVALUE %s START WITH %s CACHE %s %s;',
			namespace.nspname, relation.relname, pg_catalog.format_type(sequence_record.seqtypid, NULL),
			sequence_record.seqincrement, sequence_record.seqmin, sequence_record.seqmax, sequence_record.seqstart,
			sequence_record.seqcache, CASE WHEN sequence_record.seqcycle THEN 'CYCLE' ELSE 'NO CYCLE' END) ||
		COALESCE((SELECT pg_catalog.format(E'\nALTER SEQUENCE %I.%I OWNED BY %I.%I.%I;', namespace.nspname, relation.relname,
			target_namespace.nspname, target.relname, attribute.attname)
			FROM pg_catalog.pg_depend AS dependency JOIN pg_catalog.pg_class AS target ON target.oid = dependency.refobjid
			JOIN pg_catalog.pg_namespace AS target_namespace ON target_namespace.oid = target.relnamespace
			JOIN pg_catalog.pg_attribute AS attribute ON attribute.attrelid = target.oid AND attribute.attnum = dependency.refobjsubid
			WHERE dependency.classid = 'pg_catalog.pg_class'::regclass AND dependency.objid = relation.oid
			AND dependency.refclassid = 'pg_catalog.pg_class'::regclass AND dependency.deptype IN ('a', 'i') LIMIT 1), '')
		FROM pg_catalog.pg_sequence AS sequence_record JOIN pg_catalog.pg_class AS relation ON relation.oid = sequence_record.seqrelid
		JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = relation.relnamespace
		WHERE namespace.nspname = $1 ORDER BY relation.relname`, schema)
	if queryError != nil {
		return nil, control.SafeError(queryError)
	}
	return namedDefinitions(rows)
}
