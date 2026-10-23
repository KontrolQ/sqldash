package control

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"sqldash/models"

	"github.com/jackc/pgx/v5"
)

type Roles struct {
	Custodian   string
	Owner       string
	Reader      string
	Writer      string
	ReadAccess  string
	WriteAccess string
	Observer    string
}

func DatabaseRoles(record *models.Database) (Roles, error) {
	if record == nil || record.ID == 0 || record.Kind() != models.EnginePostgres || !validIdentifier(record.Name) {
		return Roles{}, InvalidDatabase
	}
	if record.Name == MaintenanceDatabase || record.Name == "template0" || record.Name == "template1" || strings.HasPrefix(record.Name, "pg_") {
		return Roles{}, InvalidDatabase
	}
	prefix := RolePrefix + strconv.FormatUint(uint64(record.ID), 16) + "_"
	return Roles{
		Custodian:   prefix + "custodian",
		Owner:       prefix + "owner",
		Reader:      prefix + "reader",
		Writer:      prefix + "writer",
		ReadAccess:  prefix + "read",
		WriteAccess: prefix + "write",
		Observer:    prefix + "observer",
	}, nil
}

func InternalRoleName(username string) bool {
	if !strings.HasPrefix(username, RolePrefix) {
		return false
	}
	identifier, suffix, found := strings.Cut(strings.TrimPrefix(username, RolePrefix), "_")
	if _, parseError := strconv.ParseUint(identifier, 16, 64); !found || parseError != nil {
		return false
	}
	switch suffix {
	case "custodian", "owner", "reader", "writer", "read", "write", "observer":
		return true
	}
	return strings.HasPrefix(suffix, "app_") || strings.HasPrefix(suffix, "transfer_")
}

func ApplicationUsername(record *models.Database, connection *models.PostgresConnection) (string, error) {
	if _, roleError := DatabaseRoles(record); roleError != nil {
		return "", roleError
	}
	if connection == nil || connection.DatabaseID != record.ID {
		return "", InvalidConnection
	}
	return record.Name, nil
}

func validIdentifier(value string) bool {
	return value != "" && len(value) <= MaximumIdentifierBytes && !strings.ContainsRune(value, 0) && utf8.ValidString(value)
}

func pgIdentifier(value string) string {
	return pgx.Identifier{value}.Sanitize()
}

func roleMarker(record *models.Database, name string) string {
	return fmt.Sprintf("sqldash:%d:%s", record.ID, name)
}

func databaseMarker(record *models.Database) string {
	return fmt.Sprintf("sqldash:%d", record.ID)
}

func roleList(roles Roles) []string {
	return []string{roles.Custodian, roles.Owner, roles.Reader, roles.Writer, roles.ReadAccess, roles.WriteAccess, roles.Observer}
}
