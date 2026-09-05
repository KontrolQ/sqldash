package event

import (
	store "sqldash/database"
	"sqldash/models"
)

func Create(record *models.Event) error {
	return store.DB.Create(record).Error
}

func Page(databaseName string, action string, offset int, limit int) ([]models.Event, int64, error) {
	records := make([]models.Event, 0)
	query := store.DB.Model(&models.Event{})

	if databaseName != "" {
		query = query.Where("database = ?", databaseName)
	}

	if action != "" {
		query = query.Where("action = ?", action)
	}

	var total int64

	if countError := query.Count(&total).Error; countError != nil {
		return nil, 0, countError
	}

	findError := query.Order(RecentFirst).Offset(offset).Limit(limit).Find(&records).Error
	if findError != nil {
		return nil, 0, findError
	}

	return records, total, nil
}

func Actions(databaseName string) ([]string, error) {
	held := make([]string, 0)
	query := store.DB.Model(&models.Event{})

	if databaseName != "" {
		query = query.Where("database = ?", databaseName)
	}

	if findError := query.Distinct().Order(ActionOrder).Pluck("action", &held).Error; findError != nil {
		return nil, findError
	}

	return held, nil
}
