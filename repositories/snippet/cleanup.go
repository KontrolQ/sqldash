package snippet

import (
	"sqldash/database"
	"sqldash/models"
)

func DeleteForDatabase(name string) error {
	return database.DB.Unscoped().Where("database = ?", name).Delete(&models.Snippet{}).Error
}
