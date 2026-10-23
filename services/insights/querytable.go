package insights

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"sqldash/analytics"
	"sqldash/pgmetrics"
)

func queryTableColumns() []QueryMetricColumn {
	return []QueryMetricColumn{
		{Key: ShareSort, Label: "Time share within engine", Help: "Measured time as a percentage of the selected scope within this row's engine."},
		{Key: CountSort, Label: CountHeading, Help: "Recorded libSQL statement executions or PostgreSQL calls."},
		{Key: TotalSort, Label: TotalHeading, Help: "Client duration for libSQL; native execution time for PostgreSQL."},
		{Key: "mean", Label: "Mean time", Help: "Recorded total time divided by recorded count."},
		{Key: MedianSort, Label: MedianHeading, Help: "Approximate libSQL histogram median; unavailable for native PostgreSQL statistics."},
		{Key: TailSort, Label: TailHeading, Help: "Approximate libSQL histogram P99; unavailable for native PostgreSQL statistics."},
		{Key: ReadSort, Label: ReadHeading, Help: "libSQL scanned rows, not PostgreSQL returned or affected rows."},
		{Key: RatioSort, Label: RatioHeading, Help: "libSQL scanned rows divided by returned rows. A zero denominator has no ratio."},
		{Key: WrittenSort, Label: WrittenHeading, Help: "libSQL rows written; unavailable as a separate native PostgreSQL measurement."},
		{Key: "rows", Label: "Rows processed", Help: "PostgreSQL rows returned or affected, not rows scanned."},
		{Key: "hit", Label: "Shared blocks hit", Help: "Native PostgreSQL shared block cache hits."},
		{Key: "blocks", Label: "Shared blocks read", Help: "Native PostgreSQL shared blocks read, not rows or transferred bytes."},
		{Key: "minimum", Label: "Native minimum", Help: "Sampled PostgreSQL minimum; the statistics epoch can precede this period."},
		{Key: "maximum", Label: "Native maximum", Help: "Sampled PostgreSQL maximum; the statistics epoch can precede this period."},
		{Key: "local-hit", Label: "Local blocks hit", Help: "Native PostgreSQL local block cache hits."},
		{Key: "local-read", Label: "Local blocks read", Help: "Native PostgreSQL local blocks read."},
		{Key: "temporary-read", Label: "Temporary blocks read", Help: "Native PostgreSQL temporary blocks read."},
		{Key: "temporary-written", Label: "Temporary blocks written", Help: "Native PostgreSQL temporary blocks written."},
	}
}

func libSQLTableRows(summaries []analytics.DatabaseQuerySummary) []QueryTableRow {
	totalTime := 0.0
	for _, summary := range summaries {
		totalTime += summary.TotalDuration
	}
	rows := make([]QueryTableRow, 0, len(summaries))
	for _, summary := range summaries {
		statement, fullStatement := summary.Normalised, summary.Example
		if statement == "" {
			statement = summary.Example
		}
		if fullStatement == "" {
			fullStatement = summary.Normalised
		}
		if statement == "" {
			statement = "Statement text unavailable"
		}
		row := QueryTableRow{Key: summary.Key(), SourceKeys: []string{summary.Key()}, Engine: "libsql", EngineLabel: LibSQLLabel, DatabaseID: summary.DatabaseID, Database: summary.DatabaseName, DatabasePath: queriesPathFor(summary.DatabaseName), Statement: statement, FullStatement: fullStatement, StatementLabel: "Representative stored statement. This example may precede the selected period.", metrics: make(map[string]QueryMetricCell)}
		row.metrics[CountSort] = queryIntegerCell(summary.Count)
		row.metrics[TotalSort] = queryDurationCell(summary.TotalDuration)
		row.metrics[ReadSort] = queryIntegerCell(summary.RowsRead)
		row.metrics[WrittenSort] = queryIntegerCell(summary.RowsWritten)
		if totalTime > 0 {
			share := summary.TotalDuration / totalTime * 100
			row.metrics[ShareSort] = queryNumberCell(share, fmt.Sprintf(ShareFormat, share))
		} else {
			row.metrics[ShareSort] = QueryMetricCell{Value: "Unavailable", Reason: "No measured engine runtime is available for a percentage."}
		}
		if summary.Count > 0 {
			row.metrics["mean"] = queryDurationCell(summary.Average())
		} else {
			row.metrics["mean"] = QueryMetricCell{Value: "Unavailable", Reason: "No recorded executions are available for a mean."}
		}
		if summary.Histogram.Total() > 0 {
			row.metrics[MedianSort] = queryDurationCell(summary.Histogram.Percentile(0.50))
			row.metrics[TailSort] = queryDurationCell(summary.Histogram.Percentile(0.99))
		} else {
			missingHistogram := QueryMetricCell{Value: "Unavailable", Reason: "No recorded latency histogram is available."}
			row.metrics[MedianSort], row.metrics[TailSort] = missingHistogram, missingHistogram
		}
		if summary.RowsReturned > 0 {
			ratio := float64(summary.RowsRead) / float64(summary.RowsReturned)
			row.metrics[RatioSort] = queryNumberCell(ratio, ratioOf(summary.RowsRead, summary.RowsReturned))
		} else {
			row.metrics[RatioSort] = QueryMetricCell{Value: "Unavailable", Reason: "No ratio exists when recorded returned rows are zero."}
		}
		rows = append(rows, row)
	}
	return rows
}

func nativeTableRows(summaries []pgmetrics.QuerySummary) []QueryTableRow {
	totalTime := 0.0
	for _, summary := range summaries {
		totalTime += summary.ExecutionMs
	}
	rows := make([]QueryTableRow, 0, len(summaries))
	for _, summary := range summaries {
		statement := summary.Statement
		if statement == "" {
			statement = "Statement text unavailable"
		}
		statementDigest := sha256.Sum256([]byte(summary.Statement))
		row := QueryTableRow{Key: fmt.Sprintf("%s:%x", summary.Key(), statementDigest), SourceKeys: []string{summary.Key()}, Engine: "postgres", EngineLabel: PostgresLabel, DatabaseID: summary.DatabaseID, Database: summary.DatabaseName, DatabasePath: queriesPathFor(summary.DatabaseName), Statement: statement, FullStatement: summary.Statement, StatementLabel: "Representative normalized PostgreSQL statement, not an exact original execution with bound values.", metrics: make(map[string]QueryMetricCell)}
		row.metrics[CountSort] = queryIntegerCell(summary.Calls)
		row.metrics[TotalSort] = queryDurationCell(summary.ExecutionMs)
		row.metrics["rows"] = queryIntegerCell(summary.Rows)
		row.metrics["hit"] = queryIntegerCell(summary.SharedBlocksHit)
		row.metrics["blocks"] = queryIntegerCell(summary.SharedBlocksRead)
		row.metrics["minimum"] = queryDurationCell(summary.MinimumExecutionMs)
		row.metrics["maximum"] = queryDurationCell(summary.MaximumExecutionMs)
		row.metrics["local-hit"] = queryIntegerCell(summary.LocalBlocksHit)
		row.metrics["local-read"] = queryIntegerCell(summary.LocalBlocksRead)
		row.metrics["temporary-read"] = queryIntegerCell(summary.TemporaryBlocksRead)
		row.metrics["temporary-written"] = queryIntegerCell(summary.TemporaryBlocksWritten)
		if summary.Calls > 0 {
			row.metrics["mean"] = queryDurationCell(summary.MeanExecutionMs())
		} else {
			row.metrics["mean"] = QueryMetricCell{Value: "Unavailable", Reason: "No recorded calls are available for a mean."}
		}
		if totalTime > 0 {
			share := summary.ExecutionMs / totalTime * 100
			row.metrics[ShareSort] = queryNumberCell(share, fmt.Sprintf(ShareFormat, share))
		} else {
			row.metrics[ShareSort] = QueryMetricCell{Value: "Unavailable", Reason: "No measured engine runtime is available for a percentage."}
		}
		rows = append(rows, row)
	}
	return rows
}

func queryTableSortNamed(asked string) string {
	if asked == "database" || asked == "statement" {
		return asked
	}
	for _, column := range queryTableColumns() {
		if column.Key == asked {
			return asked
		}
	}
	return TotalSort
}

func orderQueryTable(rows []QueryTableRow, by string, direction string) {
	sort.SliceStable(rows, func(first int, second int) bool {
		comparison := compareQueryRows(rows[first], rows[second], by, direction)
		if comparison == 0 {
			return rows[first].Key < rows[second].Key
		}
		return comparison < 0
	})
}

func compareQueryRows(left QueryTableRow, right QueryTableRow, by string, direction string) int {
	comparison := 0
	switch by {
	case "database":
		comparison = strings.Compare(left.Database, right.Database)
	case "statement":
		comparison = strings.Compare(left.Statement, right.Statement)
	default:
		leftCell, rightCell := left.Metric(by), right.Metric(by)
		if leftCell.Available != rightCell.Available {
			if leftCell.Available {
				return -1
			}
			return 1
		}
		if leftCell.Integer != nil && rightCell.Integer != nil {
			if *leftCell.Integer < *rightCell.Integer {
				comparison = -1
			} else if *leftCell.Integer > *rightCell.Integer {
				comparison = 1
			}
		} else if leftCell.Number != nil && rightCell.Number != nil {
			if *leftCell.Number < *rightCell.Number {
				comparison = -1
			} else if *leftCell.Number > *rightCell.Number {
				comparison = 1
			}
		}
	}
	if direction == DescendingOrder {
		return -comparison
	}
	return comparison
}

func queryControlPath(shown *QueriesContext, sortKey string, direction string, page int) string {
	values := url.Values{"window": []string{shown.Window}, "engine": []string{shown.Engine}, "sort": []string{sortKey}, "direction": []string{direction}, "page": []string{fmt.Sprint(page)}}
	return shown.QueriesPath + "?" + values.Encode()
}

func paginateQueryTable(shown *QueriesContext, rows []QueryTableRow, askedPage int) {
	shown.Total = int64(len(rows))
	shown.PageSize = QueryPageSize
	shown.Pages = pagesFor(len(rows), QueryPageSize)
	shown.Page = max(1, min(askedPage, max(1, shown.Pages)))
	from := min((shown.Page-1)*QueryPageSize, len(rows))
	to := min(from+QueryPageSize, len(rows))
	shown.TableRows = rows[from:to]
	shown.TableColumns = queryTableColumns()
	for index := range shown.TableColumns {
		column := &shown.TableColumns[index]
		column.Path = queryControlPath(shown, column.Key, nextQueryDirection(shown, column.Key), 1)
	}
	for index := range shown.TableRows {
		row := &shown.TableRows[index]
		identifier := sha256.Sum256([]byte(row.Key))
		row.StatementID = fmt.Sprintf("query-statement-%x", identifier)
		row.Cells = make([]QueryMetricCell, 0, len(shown.TableColumns))
		for _, column := range shown.TableColumns {
			row.Cells = append(row.Cells, row.Metric(column.Key))
		}
	}
	if len(shown.TableRows) > 0 {
		shown.FirstRow, shown.LastRow = from+1, to
	}
	shown.PreviousPage, shown.NextPage = shown.Page-1, shown.Page+1
	shown.FirstPagePath = queryControlPath(shown, shown.Sort, shown.Direction, 1)
	shown.PreviousPagePath = queryControlPath(shown, shown.Sort, shown.Direction, max(1, shown.PreviousPage))
	shown.NextPagePath = queryControlPath(shown, shown.Sort, shown.Direction, min(max(1, shown.Pages), shown.NextPage))
	shown.LastPagePath = queryControlPath(shown, shown.Sort, shown.Direction, max(1, shown.Pages))
	shown.DatabaseSortPath = queryControlPath(shown, "database", nextQueryDirection(shown, "database"), 1)
	shown.StatementSortPath = queryControlPath(shown, "statement", nextQueryDirection(shown, "statement"), 1)
	shown.FilterQuery = url.Values{"sort": []string{shown.Sort}, "direction": []string{shown.Direction}}.Encode()
}

func nextQueryDirection(shown *QueriesContext, key string) string {
	if shown.Sort == key && shown.Direction == DescendingOrder {
		return AscendingOrder
	}
	return DescendingOrder
}
