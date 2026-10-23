package engines

import (
	"context"
	"errors"
	"sqldash/database"
	"sqldash/models"
	"sync"
)

var registry = struct {
	sync.RWMutex
	backends map[string]Backend
}{backends: map[string]Backend{models.EngineLibSQL: libSQLBackend{}}}

func Register(kind string, backend Backend) {
	registry.Lock()
	defer registry.Unlock()
	registry.backends[kind] = backend
}

func For(record *models.Database) (Backend, error) {
	if record == nil {
		return nil, DatabaseUnavailable
	}
	registry.RLock()
	defer registry.RUnlock()
	backend, known := registry.backends[record.Kind()]
	if !known {
		return nil, UnknownEngine
	}
	return backend, nil
}

func Resolve(parent context.Context, name string) (*models.Database, error) {
	if record := CurrentDatabase(parent); record != nil && record.Name == name {
		if !record.Available() {
			return nil, DatabaseUnavailable
		}
		return record, nil
	}
	var record models.Database
	if findError := database.DB.WithContext(parent).Where("name = ?", name).First(&record).Error; findError != nil {
		return nil, findError
	}
	if !record.Available() {
		return nil, DatabaseUnavailable
	}
	return &record, nil
}

func Query(parent context.Context, name string, statement string, arguments ...any) (*Result, error) {
	results, runError := Run(parent, name, Statement{SQL: statement, Arguments: arguments})
	if runError != nil {
		return nil, runError
	}
	if len(results) == 0 {
		return nil, DatabaseUnavailable
	}
	return results[0], nil
}

func Run(parent context.Context, name string, statements ...Statement) ([]*Result, error) {
	record, findError := Resolve(parent, name)
	if findError != nil {
		return nil, findError
	}
	backend, findError := For(record)
	if findError != nil {
		return nil, findError
	}
	results, runError := backend.Run(parent, record, CurrentMode(parent), statements...)
	if errors.Is(runError, PolicyUnconfirmed) {
		if quarantineError := Quarantine(parent, record); quarantineError != nil {
			return nil, errors.Join(runError, quarantineError)
		}
	}
	return results, runError
}

func Statistics(parent context.Context, name string) (*StatisticsRecord, error) {
	record, findError := Resolve(parent, name)
	if findError != nil {
		return nil, findError
	}
	backend, findError := For(record)
	if findError != nil {
		return nil, findError
	}
	return backend.Statistics(parent, record)
}

func Configuration(parent context.Context, name string) (*ConfigurationRecord, error) {
	record, findError := Resolve(parent, name)
	if findError != nil {
		return nil, findError
	}
	backend, findError := For(record)
	if findError != nil {
		return nil, findError
	}
	return backend.Configuration(parent, record)
}

func SetConfiguration(parent context.Context, name string, wanted ConfigurationRecord) error {
	record, findError := Resolve(parent, name)
	if findError != nil {
		return findError
	}
	backend, findError := For(record)
	if findError != nil {
		return findError
	}
	return backend.SetConfiguration(parent, record, wanted)
}
