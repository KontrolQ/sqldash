package hrana

type Statement struct {
	SQL          string
	DurationMs   float64
	RowsRead     int64
	RowsWritten  int64
	RowsReturned int64
	Failed       bool
}

type askedPipeline struct {
	Requests []askedRequest `json:"requests"`
}

type askedRequest struct {
	Type  string      `json:"type"`
	Stmt  *askedStmt  `json:"stmt"`
	Stmts []askedStmt `json:"stmts"`
}

type askedStmt struct {
	SQL string `json:"sql"`
}

type heldPipeline struct {
	Results []heldResult `json:"results"`
}

type heldResult struct {
	Type     string        `json:"type"`
	Response *heldResponse `json:"response"`
}

type heldResponse struct {
	Type        string         `json:"type"`
	Result      *heldResult2   `json:"result"`
	StepResults []*heldResult2 `json:"step_results"`
}

type heldResult2 struct {
	Rows        []any   `json:"rows"`
	RowsRead    int64   `json:"rows_read"`
	RowsWritten int64   `json:"rows_written"`
	DurationMs  float64 `json:"query_duration_ms"`
}
