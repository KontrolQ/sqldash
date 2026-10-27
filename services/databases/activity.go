package databases

import (
	"time"

	"sqldash/engines"
	"sqldash/models"
	"sqldash/pgcluster/control"
	"sqldash/redishost"
)

func unavailableActivityMetrics(engine string) ActivityMetrics {
	metrics := ActivityMetrics{RowsRead: "Unavailable", QueryCount: "Unavailable", RowsReadLabel: "Rows read", QueryCountLabel: "Queries", RowsReadNote: "Cumulative rows read from the libSQL engine's native statistics.", QueryCountNote: "Cumulative queries from the libSQL engine's native statistics."}
	switch engine {
	case models.EnginePostgres:
		metrics.RowsReadLabel, metrics.QueryCountLabel = "Tuples returned", "Statement calls"
		metrics.RowsReadNote = "PostgreSQL live rows from sequential scans and index entries from index scans, since database statistics were reset. Not SQL result rows."
		metrics.QueryCountNote = "Tracked top-level calls for this database's application roles, since statement statistics were reset or evicted. Sensitive statements are excluded; unavailable without statement statistics."
	case models.EngineRedis:
		metrics.RowsReadLabel, metrics.QueryCountLabel = "Keyspace hits", "Commands"
		metrics.RowsReadNote = "Successful Redis keyspace lookups since server startup or statistics reset. Not SQL rows."
		metrics.QueryCountNote = "Redis commands processed since server startup or statistics reset, including monitoring commands."
	}
	return metrics
}

func libsqlActivityMetrics(statistics *engines.StatisticsRecord) ActivityMetrics {
	metrics := unavailableActivityMetrics(models.EngineLibSQL)
	metrics.RowsRead = readableCount(statistics.RowsRead)
	metrics.QueryCount = readableCount(statistics.QueryCount)
	return metrics
}

func postgresActivityMetrics(statistics *control.StatisticsRecord) ActivityMetrics {
	metrics := unavailableActivityMetrics(models.EnginePostgres)
	metrics.RowsRead = readableCount(statistics.TuplesReturned)
	if statistics.ResetAt != nil {
		metrics.RowsReadNote += " Last reset: " + statistics.ResetAt.UTC().Format(time.RFC3339) + "."
	}
	if statistics.StatementsAvailable {
		metrics.QueryCount = readableCount(statistics.StatementCalls)
	}
	if statistics.StatementResetAt != nil {
		metrics.QueryCountNote += " Last global reset: " + statistics.StatementResetAt.UTC().Format(time.RFC3339) + "."
	}
	return metrics
}

func redisActivityMetrics(statistics *redishost.Statistics) ActivityMetrics {
	metrics := unavailableActivityMetrics(models.EngineRedis)
	metrics.RowsRead = readableCount(statistics.KeyspaceHits)
	metrics.QueryCount = readableCount(statistics.CommandsProcessed)
	return metrics
}
