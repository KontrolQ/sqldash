package insights

const (
	LogPrefix = "Insights"

	OverviewTitle = "Insights"
	QueriesTitle  = "Top queries"
	TopQueryLimit = 20
	QueryPageSize = 10

	ShareSort   = "share"
	CountSort   = "count"
	TotalSort   = "total"
	MedianSort  = "median"
	TailSort    = "tail"
	ReadSort    = "read"
	RatioSort   = "ratio"
	WrittenSort = "written"

	ShareHeading   = "% of runtime"
	CountHeading   = "Count"
	TotalHeading   = "Total time"
	MedianHeading  = "P50"
	TailHeading    = "P99"
	ReadHeading    = "Rows read"
	RatioHeading   = "Rows read / rows returned"
	WrittenHeading = "Rows written"

	AscendingOrder  = "asc"
	DescendingOrder = "desc"

	MillisecondSuffix    = " ms"
	SecondSuffix         = " sec"
	ShareFormat          = "%.1f%%"
	RatioFormat          = "%.1f"
	NoRatio              = "—"
	MillisecondsInSecond = 1000.0

	Thousand = 1_000.0
	Million  = 1_000_000.0
	Billion  = 1_000_000_000.0

	ThousandSuffix = "k"
	MillionSuffix  = "M"
	BillionSuffix  = "B"

	CompactFormat = "%.2f"
	TightFormat   = "%.1f"
	TightAbove    = 10.0
	RiseFormat    = "+%.1f%%"
	FallFormat    = "-%.1f%%"
	SteadyChange  = "no change"
	NewChange     = "new"

	ChangeUp     = "up"
	ChangeDown   = "down"
	ChangeSteady = "steady"
)

const (
	EveryDatabase = "every database"
	LastPrefix    = "Last "
)

const (
	OverviewPath   = "/insights"
	QueriesPath    = "/insights/queries"
	DatabasePath   = "/databases/"
	InsightsSuffix = "/insights"
	QueriesSuffix  = "/queries"
)

const (
	ChartWidth  = 1000.0
	ChartHeight = 160.0
	BarFill     = 0.72

	NumberFormat   = "%.2f"
	PointFormat    = "%s%.2f %.2f "
	MoveTo         = "M"
	LineTo         = "L"
	BarLabelFormat = "%d queries · %d read · %d written · %s"
	MomentLayout   = "Jan 2, 15:04"
	TickLayout     = "Jan 2, 15:04"
	RowsAxis       = "rows"

	StackedKind     = "stacked"
	LineKind        = "line"
	MillisecondUnit = "ms"
	ReadsName       = "Reads"
	WritesName      = "Writes"

	MoreIsBetter    = "more"
	LessIsBetter    = "less"
	NeitherIsBetter = "neither"

	GoodTone  = "good"
	BadTone   = "bad"
	PlainTone = "plain"

	DurationScale = 1000.0

	SparkWidth   = 120.0
	SparkHeight  = 32.0
	SparkPadding = 4.0
	FillFormat   = "%s L%.2f %.2f L0 %.2f Z"

	QueriesLabel     = "Queries"
	FailuresLabel    = "Failures"
	RowsReadLabel    = "Rows read"
	RowsWrittenLabel = "Rows written"
	AverageLabel     = "Average"
	TailLabel        = "P99"

	P50Name = "P50"
	P95Name = "P95"
	P99Name = "P99"
)
