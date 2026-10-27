package control

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"sqldash/config"
	"sqldash/models"
	"sqldash/secretstore"

	"github.com/jackc/pgx/v5"
)

func acquireDatabaseLock(ctx context.Context, connection *pgx.Conn, record *models.Database) error {
	_, lockError := connection.Exec(ctx, "SELECT pg_catalog.pg_advisory_lock(hashtextextended($1, 0))", databaseMarker(record))
	return SafeError(lockError)
}

func ensureRole(ctx context.Context, transaction pgx.Tx, record *models.Database, name string, login bool, password string) error {
	var marker string
	queryError := transaction.QueryRow(ctx, `SELECT COALESCE(pg_catalog.shobj_description(oid, 'pg_authid'), '')
		FROM pg_catalog.pg_roles WHERE rolname = $1`, name).Scan(&marker)
	if errors.Is(queryError, pgx.ErrNoRows) {
		if _, createError := transaction.Exec(ctx, "CREATE ROLE "+pgIdentifier(name)); createError != nil {
			return SafeError(createError)
		}
		if _, commentError := transaction.Exec(ctx, "COMMENT ON ROLE "+pgIdentifier(name)+" IS "+pgLiteral(roleMarker(record, name))); commentError != nil {
			return SafeError(commentError)
		}
	} else if queryError != nil {
		return SafeError(queryError)
	} else if marker != roleMarker(record, name) {
		return RoleMismatch
	}
	loginOption := "NOLOGIN PASSWORD NULL"
	if login {
		loginOption = "LOGIN"
		if password != "" {
			verifier, verifierError := passwordVerifier(password)
			if verifierError != nil {
				return verifierError
			}
			loginOption += " PASSWORD " + pgLiteral(verifier)
		}
	}
	command := "ALTER ROLE " + pgIdentifier(name) + " WITH NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS INHERIT " + loginOption + " VALID UNTIL 'infinity' CONNECTION LIMIT -1"
	if _, alterError := transaction.Exec(ctx, command); alterError != nil {
		return SafeError(alterError)
	}
	rows, queryError := transaction.Query(ctx, `SELECT parent.rolname FROM pg_catalog.pg_auth_members AS membership
		JOIN pg_catalog.pg_roles AS parent ON parent.oid = membership.roleid
		JOIN pg_catalog.pg_roles AS member ON member.oid = membership.member WHERE member.rolname = $1`, name)
	if queryError != nil {
		return SafeError(queryError)
	}
	parents := []string{}
	for rows.Next() {
		var parent string
		if scanError := rows.Scan(&parent); scanError != nil {
			rows.Close()
			return SafeError(scanError)
		}
		parents = append(parents, parent)
	}
	queryError = rows.Err()
	rows.Close()
	if queryError != nil {
		return SafeError(queryError)
	}
	for _, parent := range parents {
		if _, revokeError := transaction.Exec(ctx, "REVOKE "+pgIdentifier(parent)+" FROM "+pgIdentifier(name)+" CASCADE"); revokeError != nil {
			return SafeError(revokeError)
		}
	}
	if _, resetError := transaction.Exec(ctx, "ALTER ROLE "+pgIdentifier(name)+" RESET ALL"); resetError != nil {
		return SafeError(resetError)
	}
	return nil
}

func dataMembership(ctx context.Context, transaction pgx.Tx, member, group string) error {
	_, grantError := transaction.Exec(ctx, "GRANT "+pgIdentifier(group)+" TO "+pgIdentifier(member)+" WITH INHERIT TRUE, SET FALSE, ADMIN FALSE")
	return SafeError(grantError)
}

func internalPassword(store *secretstore.Store, username string, create bool) (string, error) {
	var password []byte
	var passwordError error
	if create {
		password, passwordError = store.GetOrCreate(SecretNamespace, username, func() ([]byte, error) {
			generated, generateError := GeneratePassword()
			return []byte(generated), generateError
		})
	} else {
		password, passwordError = store.Get(SecretNamespace, username)
	}
	if passwordError != nil {
		return "", Unavailable
	}
	defer clear(password)
	if len(password) == 0 || strings.ContainsRune(string(password), 0) {
		return "", Unavailable
	}
	return string(password), nil
}

func provisionRole(ctx context.Context, transaction pgx.Tx, store *secretstore.Store, record *models.Database, name string, login bool) error {
	password := ""
	if login {
		var marker string
		findError := transaction.QueryRow(ctx, `SELECT COALESCE(pg_catalog.shobj_description(oid, 'pg_authid'), '') FROM pg_catalog.pg_roles WHERE rolname = $1`, name).Scan(&marker)
		create := errors.Is(findError, pgx.ErrNoRows)
		if findError != nil && !create {
			return SafeError(findError)
		}
		if !create && marker != roleMarker(record, name) {
			return RoleMismatch
		}
		var passwordError error
		password, passwordError = internalPassword(store, name, create)
		if passwordError != nil {
			return passwordError
		}
	}
	return ensureRole(ctx, transaction, record, name, login, password)
}

func provisionRoles(ctx context.Context, connection *pgx.Conn, record *models.Database, roles Roles) error {
	store, storeError := secretstore.Open(config.PostgresSecretsPath())
	if storeError != nil {
		return Unavailable
	}
	transaction, beginError := connection.Begin(ctx)
	if beginError != nil {
		return SafeError(beginError)
	}
	defer rollbackTransaction(transaction)
	for _, name := range roleList(roles) {
		login := name == roles.Owner || name == roles.Reader || name == roles.Writer || name == roles.Observer
		if ensureError := provisionRole(ctx, transaction, store, record, name, login); ensureError != nil {
			return ensureError
		}
	}
	for member, group := range map[string]string{roles.Reader: roles.ReadAccess, roles.Writer: roles.WriteAccess, roles.Owner: roles.ObjectAccess} {
		if grantError := dataMembership(ctx, transaction, member, group); grantError != nil {
			return grantError
		}
	}
	for _, name := range []string{roles.Owner, roles.Reader, roles.Writer} {
		readOnly := "off"
		if name == roles.Reader {
			readOnly = "on"
		}
		for setting, value := range map[string]string{
			"default_transaction_read_only":       readOnly,
			"statement_timeout":                   "0",
			"idle_in_transaction_session_timeout": "0",
			"lock_timeout":                        "0",
		} {
			if _, settingError := transaction.Exec(ctx, fmt.Sprintf("ALTER ROLE %s SET %s = %s", pgIdentifier(name), setting, pgLiteral(value))); settingError != nil {
				return SafeError(settingError)
			}
		}
	}
	if configureError := configureMetadataRole(ctx, transaction, roles); configureError != nil {
		return configureError
	}
	return SafeError(transaction.Commit(ctx))
}
