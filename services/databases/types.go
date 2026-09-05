package databases

import (
	tokenservice "sqldash/services/tokens"
	"sqldash/utils/collections"
)

type IndexContext struct {
	Title     string
	Databases []DatabaseView
}

type DatabaseView struct {
	Name      string
	Address   string
	Protected bool
	Blocked   bool
	Storage   string
	Tables    int
	RowsRead  string
	Queries   string
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
	Tables      int
	RowsRead    string
	RowsWritten string
	QueryCount  string
	Tokens      []tokenservice.TokenView
	Secret      string
	Scopes      []collections.Option
	Rules       []RuleView
}

type RuleView struct {
	Identifier uint
	Network    string
	Note       string
}
