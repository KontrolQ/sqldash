package access

import (
	"context"
	"sqldash/permissions"
)

func MonitoringScope(parent context.Context) ([][]string, error) {
	return MonitoringScopeFor(parent, permissions.InsightsView)
}

func MonitoringScopeFor(parent context.Context, permission string) ([][]string, error) {
	records, findError := VisibleFor(FromContext(parent), permission)
	if findError != nil {
		return nil, findError
	}
	names := make([]string, 0, len(records))
	for _, record := range records {
		names = append(names, record.Name)
	}
	return [][]string{names}, nil
}
