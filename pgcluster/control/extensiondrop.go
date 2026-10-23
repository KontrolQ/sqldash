package control

import (
	"context"

	"sqldash/config"

	"github.com/jackc/pgx/v5"
)

const (
	ExtensionDropGuard         = "sqd_extension_drop_guard"
	ExtensionDropGuardFunction = "guard_extension_drops"
	ExtensionDropGuardMarker   = "sqldash:extension-drop-guard"
)

func guardExtensionDrops(ctx context.Context, transaction pgx.Tx, recipients string) error {
	var exists bool
	var marker string
	queryError := transaction.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_event_trigger WHERE evtname = $1),
		COALESCE((SELECT pg_catalog.obj_description(oid, 'pg_event_trigger') FROM pg_catalog.pg_event_trigger WHERE evtname = $1), '')`, ExtensionDropGuard).Scan(&exists, &marker)
	if queryError != nil {
		return SafeError(queryError)
	}
	if exists && marker != ExtensionDropGuardMarker {
		return IdentityMismatch
	}
	function := pgx.Identifier{ControlSchema, ExtensionDropGuardFunction}.Sanitize()
	command := "CREATE OR REPLACE FUNCTION " + function + "() RETURNS event_trigger LANGUAGE plpgsql SET search_path = pg_catalog AS $sqd$ BEGIN IF session_user <> " + pgLiteral(config.Postgres.AdminUsername) + " AND EXISTS (SELECT 1 FROM pg_catalog.pg_event_trigger_dropped_objects() WHERE object_type = 'extension') THEN RAISE EXCEPTION 'extension objects cannot be removed by SQL sessions' USING ERRCODE = '42501'; END IF; END; $sqd$"
	if _, createError := transaction.Exec(ctx, command); createError != nil {
		return SafeError(createError)
	}
	if _, revokeError := transaction.Exec(ctx, "REVOKE ALL ON FUNCTION "+function+"() FROM "+recipients); revokeError != nil {
		return SafeError(revokeError)
	}
	if !exists {
		if _, createError := transaction.Exec(ctx, "CREATE EVENT TRIGGER "+pgIdentifier(ExtensionDropGuard)+" ON sql_drop EXECUTE FUNCTION "+function+"()"); createError != nil {
			return SafeError(createError)
		}
		if _, markerError := transaction.Exec(ctx, "COMMENT ON EVENT TRIGGER "+pgIdentifier(ExtensionDropGuard)+" IS "+pgLiteral(ExtensionDropGuardMarker)); markerError != nil {
			return SafeError(markerError)
		}
	} else {
		var correct bool
		if verifyError := transaction.QueryRow(ctx, `SELECT trigger_record.evtfoid = routine.oid FROM pg_catalog.pg_event_trigger AS trigger_record
			JOIN pg_catalog.pg_proc AS routine ON routine.proname = $2 JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid = routine.pronamespace
			WHERE trigger_record.evtname = $1 AND namespace.nspname = $3`, ExtensionDropGuard, ExtensionDropGuardFunction, ControlSchema).Scan(&correct); verifyError != nil {
			return SafeError(verifyError)
		}
		if !correct {
			return IdentityMismatch
		}
	}
	_, enableError := transaction.Exec(ctx, "ALTER EVENT TRIGGER "+pgIdentifier(ExtensionDropGuard)+" ENABLE ALWAYS")
	return SafeError(enableError)
}
