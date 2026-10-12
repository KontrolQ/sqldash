package access

import "context"

func MonitoringScope(parent context.Context) ([][]string, error) {
	held := FromContext(parent)
	if held != nil && held.All() {
		return nil, nil
	}
	records, findError := Visible(held)
	if findError != nil {
		return nil, findError
	}
	names := make([]string, 0, len(records))
	for _, record := range records {
		names = append(names, record.Name)
	}
	return [][]string{names}, nil
}
