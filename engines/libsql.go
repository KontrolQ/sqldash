package engines

import (
	"context"
	"errors"
	"net/http"
	"sqldash/models"
	"sqldash/sqld"
	"time"
)

type libSQLBackend struct{}

func (self libSQLBackend) Features() Features {
	return Features{SQLiteFiles: true, HistoricalFork: true, Attach: true, ReadWriteBlocks: true, MeasuredQueries: true}
}

func (self libSQLBackend) Create(parent context.Context, record *models.Database) error {
	return sqld.Create(parent, record.Name)
}

func (self libSQLBackend) Delete(parent context.Context, record *models.Database) error {
	deleteError := sqld.Delete(parent, record.Name)
	var response *sqld.AdminError
	if errors.As(deleteError, &response) && response.Status == http.StatusNotFound {
		return nil
	}
	return deleteError
}

func (self libSQLBackend) Run(parent context.Context, record *models.Database, mode Mode, statements ...Statement) ([]*Result, error) {
	return sqld.Run(parent, record.Name, statements...)
}

func (self libSQLBackend) Statistics(parent context.Context, record *models.Database) (*StatisticsRecord, error) {
	return sqld.Statistics(parent, record.Name)
}

func (self libSQLBackend) Configuration(parent context.Context, record *models.Database) (*ConfigurationRecord, error) {
	return sqld.Configuration(parent, record.Name)
}

func (self libSQLBackend) SetConfiguration(parent context.Context, record *models.Database, wanted ConfigurationRecord) error {
	return sqld.SetConfiguration(parent, record.Name, wanted)
}

func (self libSQLBackend) Fork(parent context.Context, source *models.Database, target *models.Database, moment *time.Time) error {
	return sqld.Fork(parent, source.Name, target.Name, moment)
}
