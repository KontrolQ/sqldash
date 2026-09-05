package insights

import "sqldash/utils/collections"

type OverviewContext struct {
	Title         string
	Heading       string
	FormAction    string
	Scope         string
	Window        string
	Windows       []string
	WindowOptions []collections.Option
	Queries       int64
	Failures      int64
	RowsRead      int64
	RowsWritten   int64
	Average       string
	P50           string
	P75           string
	P90           string
	P95           string
	P99           string
	Top           []QueryView
	TrafficChart  string
	LatencyChart  string
	TrafficSeries []string
	LatencySeries []string
	Approximate   bool
}

type QueryView struct {
	Statement   string
	Count       int64
	TotalTime   string
	Share       string
	P50         string
	P99         string
	RowsRead    int64
	RowsWritten int64
	ReadPerRow  string
}
