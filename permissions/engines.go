package permissions

import "strings"

func ForEngine(key string, engine string) string {
	if !strings.HasPrefix(key, "libsql.") && !strings.HasPrefix(key, "postgres.") {
		return key
	}
	if engine == "" || engine == "libsql" {
		if strings.HasPrefix(key, "libsql.") {
			return key
		}
		return ""
	}
	if engine != "postgres" {
		return ""
	}
	if key == TokenManage {
		return PostgresCredentialsManage
	}
	if strings.HasPrefix(key, "postgres.") {
		return key
	}
	mapped := "postgres." + strings.TrimPrefix(key, "libsql.")
	if !Known(mapped) {
		return ""
	}
	return mapped
}

func Dependencies(key string) []string {
	switch key {
	case SQLExecute, TokenManage:
		return []string{DataWrite}
	case PostgresDataWrite, PostgresSQLExecute:
		return []string{PostgresDataRead}
	case PostgresObjectsManage, PostgresCredentialsManage:
		return []string{PostgresDataRead, PostgresDataWrite}
	case PostgresActivityManage:
		return []string{PostgresActivityView}
	}
	return nil
}

func Satisfied(grants map[string]bool, key string) bool {
	if key == "" || !grants[key] {
		return false
	}
	for _, dependency := range Dependencies(key) {
		if !Satisfied(grants, dependency) {
			return false
		}
	}
	return true
}
