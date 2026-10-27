package redis

import "sqldash/config"

func consoleReplyLimits() (int, int) {
	budget, elements := -1, -1
	if config.Query.MaximumResultBytes > 0 {
		budget = int(config.Query.MaximumResultBytes)
	}
	if config.Query.MaximumRows > 0 {
		elements = config.Query.MaximumRows
	}
	return budget, elements
}

func consumeReplyBytes(budget *int, count int) {
	if *budget >= 0 {
		*budget = max(0, *budget-count)
	}
}
