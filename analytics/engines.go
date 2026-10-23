package analytics

import "gorm.io/gorm"

func currentLibSQL(query *gorm.DB, table string) *gorm.DB {
	return query.Where("EXISTS (SELECT 1 FROM databases WHERE databases.id = " + table + ".database_id AND databases.name = " + table + ".database_name AND databases.deleted_at IS NULL AND databases.engine IN ('', 'libsql') AND databases.state IN ('', 'ready'))")
}
