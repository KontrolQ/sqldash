package engines

import (
	"context"
	"sqldash/models"
)

type modeKey struct{}
type databaseKey struct{}

func WithMode(parent context.Context, mode Mode) context.Context {
	return context.WithValue(parent, modeKey{}, mode)
}

func CurrentMode(parent context.Context) Mode {
	if held, valid := parent.Value(modeKey{}).(Mode); valid {
		return held
	}
	return ReadOnly
}

func SQLMode(writeAllowed bool, objectsAllowed bool) Mode {
	if !writeAllowed {
		return ReadOnly
	}
	if objectsAllowed {
		return Objects
	}
	return ReadWrite
}

func WithDatabase(parent context.Context, record *models.Database) context.Context {
	return context.WithValue(parent, databaseKey{}, record)
}

func CurrentDatabase(parent context.Context) *models.Database {
	record, _ := parent.Value(databaseKey{}).(*models.Database)
	return record
}
