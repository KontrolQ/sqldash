package explorer

import (
	"net/url"
	"strconv"
)

func carryOf(table string, search string, filters []FilterView) string {
	carried := url.Values{}
	carried.Set(TableKey, table)

	if search != "" {
		carried.Set(SearchKey, search)
	}

	for _, filter := range filters {
		carried.Add(ColumnKey, filter.Column)
		carried.Add(OperatorKey, filter.Operator)
		carried.Add(ValueKey, filter.Value)
	}

	return carried.Encode()
}

func withoutSearch(table string, filters []FilterView) string {
	return carryOf(table, "", filters)
}

func withoutFilter(table string, search string, filters []FilterView, dropped int) string {
	kept := make([]FilterView, 0, len(filters))

	for index, filter := range filters {
		if index != dropped {
			kept = append(kept, filter)
		}
	}

	return carryOf(table, search, kept)
}

func viewOf(shown *BrowseContext) string {
	carried, parseError := url.ParseQuery(shown.Carry)
	if parseError != nil {
		return shown.Carry
	}

	carried.Set(PageKey, strconv.Itoa(shown.Page))

	if shown.Sort != "" {
		carried.Set(SortKey, shown.Sort)
		carried.Set(DirectionKey, shown.Direction)
	}

	return carried.Encode()
}
