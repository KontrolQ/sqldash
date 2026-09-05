package insights

import "sqldash/utils/collections"

type OverviewContext struct {
	Title         string
	Heading       string
	FormAction    string
	QueriesPath   string
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
	Tiles         []TileView
	Top           []QueryView
	TrafficChart  string
	LatencyChart  string
	TrafficSeries []string
	LatencySeries []string
	Approximate   bool
}

type QueryView struct {
	Statement   string
	Example     string
	Count       string
	TotalTime   string
	Share       string
	P50         string
	P99         string
	RowsRead    string
	RowsWritten string
	ReadPerRow  string
}

type TileView struct {
	Label  string
	Value  string
	Change string
	Kind   string
	Tone   string
	Alarm  bool
	Spark  string
	Fill   string
}

type QueryColumn struct {
	Key     string
	Label   string
	Numeric bool
}

type QueriesContext struct {
	Title         string
	Heading       string
	Scope         string
	Database      string
	FormAction    string
	QueriesPath   string
	Window        string
	WindowOptions []collections.Option
	Sort          string
	Direction     string
	Columns       []QueryColumn
	Rows          []QueryView
	Total         int64
	Page          int
	Pages         int
	PageSize      int
	PreviousPage  int
	NextPage      int
	FirstRow      int
	LastRow       int
}
