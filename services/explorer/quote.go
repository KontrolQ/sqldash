package explorer

import "strings"

func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func known(tables []TableView, name string) bool {
	for _, table := range tables {
		if table.Name == name {
			return true
		}
	}

	return false
}

func directionOf(asked string) string {
	if strings.EqualFold(asked, DescendingOrder) {
		return DescendingOrder
	}

	return AscendingOrder
}
