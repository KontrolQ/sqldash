package databases

import (
	tokenservice "sqldash/services/tokens"
)

type IndexContext struct {
	Title     string
	Databases []DatabaseView
	Problem   string
}

type DatabaseView struct {
	Name      string
	Address   string
	Protected bool
}

type ShowContext struct {
	Title       string
	Name        string
	Address     string
	URL         string
	Protected   bool
	BlockReads  bool
	BlockWrites bool
	BlockReason string
	AllowAttach bool
	Storage     string
	RowsRead    int64
	RowsWritten int64
	QueryCount  int64
	Tokens      []tokenservice.TokenView
	Secret      string
	Problem     string
	Done        string
}
