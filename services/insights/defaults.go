package insights

const (
	LogPrefix = "Insights"

	OverviewTitle = "Insights"
	TopQueryLimit = 20

	MillisecondFormat    = "%.2f ms"
	SecondFormat         = "%.2f s"
	ShareFormat          = "%.1f%%"
	RatioFormat          = "%.1f"
	NoRatio              = "—"
	MillisecondsInSecond = 1000.0
)

const (
	EveryDatabase = "every database"
	LastPrefix    = "Last "
)

const (
	OverviewPath   = "/insights"
	DatabasePath   = "/databases/"
	InsightsSuffix = "/insights"
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
	MomentLayout   = "2 Jan 15:04"
	TickLayout     = "2 Jan 15:04"

	StackedKind     = "stacked"
	LineKind        = "line"
	MillisecondUnit = "ms"
	ReadsName       = "Reads"
	WritesName      = "Writes"

	P50Name = "P50"
	P95Name = "P95"
	P99Name = "P99"
)
