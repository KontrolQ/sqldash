package permissions

import "strings"

func ForEngine(key string, engine string) string {
	prefix, _, separated := strings.Cut(key, ".")
	if !separated || prefix != "libsql" && prefix != "postgres" && prefix != "redis" {
		return key
	}
	if engine == "" {
		engine = "libsql"
	}
	if prefix == engine {
		if Known(key) {
			return key
		}
		return ""
	}
	if prefix != "libsql" {
		return ""
	}
	if engine != "postgres" && engine != "redis" {
		return ""
	}
	if key == TokenManage {
		if engine == "redis" {
			return RedisCredentialsManage
		}
		return PostgresCredentialsManage
	}
	if key == SQLExecute && engine == "redis" {
		return RedisConsoleExecute
	}
	mapped := engine + "." + strings.TrimPrefix(key, "libsql.")
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
	case RedisDataWrite, RedisConsoleExecute, RedisCredentialsManage, RedisSlowLogView, RedisMonitorView:
		return []string{RedisDataRead}
	case RedisActivityManage:
		return []string{RedisActivityView}
	case RedisSlowLogManage:
		return []string{RedisSlowLogView}
	case RedisSubscribe, RedisPublish:
		return []string{RedisDatabaseView}
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
