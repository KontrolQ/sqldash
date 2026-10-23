package event

import (
	"gorm.io/gorm"
	"sqldash/database"
	"sqldash/models"
)

type Scope struct {
	DatabaseIDs  []uint
	Engines      []string
	AllDatabases bool
	Platform     bool
	Engine       string
	DatabaseID   uint
}

func Scoped(scope Scope, databaseName string, action string) *gorm.DB {
	query := database.DB.Model(&models.Event{}).Where(
		"(engine IN ? AND (? OR database_id IN ?)) OR (? AND database_id = 0 AND database = '' AND engine = '')",
		scope.Engines, scope.AllDatabases, scope.DatabaseIDs, scope.Platform,
	)
	if scope.Engine == "platform" {
		query = query.Where("database_id = 0 AND database = '' AND engine = ''")
	} else if scope.Engine != "" {
		query = query.Where("engine = ?", scope.Engine)
	}
	if scope.DatabaseID != 0 {
		query = query.Where("database_id = ?", scope.DatabaseID)
	} else if databaseName != "" {
		query = query.Where("database = ?", databaseName)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	return query
}

func PageScoped(scope Scope, databaseName string, action string, offset int, limit int) ([]models.Event, int64, error) {
	query := Scoped(scope, databaseName, action)
	var total int64
	if findError := query.Count(&total).Error; findError != nil {
		return nil, 0, findError
	}
	var records []models.Event
	findError := query.Order(RecentFirst).Offset(offset).Limit(limit).Find(&records).Error
	return records, total, findError
}

func ActionsScoped(scope Scope, databaseName string) ([]string, error) {
	var actions []string
	findError := Scoped(scope, databaseName, "").Distinct().Order(ActionOrder).Pluck("action", &actions).Error
	return actions, findError
}
