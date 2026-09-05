package accessrule

import (
	store "sqldash/database"
	"sqldash/models"
)

func ForDatabase(name string) ([]models.AccessRule, error) {
	records := make([]models.AccessRule, 0)

	if findError := store.DB.Where("database_name = ?", name).Find(&records).Error; findError != nil {
		return nil, findError
	}

	return records, nil
}

func Create(record *models.AccessRule) error {
	return store.DB.Create(record).Error
}

func Delete(record *models.AccessRule) error {
	return store.DB.Unscoped().Delete(record).Error
}

func DeleteForDatabase(name string) error {
	return store.DB.Unscoped().Where("database_name = ?", name).Delete(&models.AccessRule{}).Error
}
