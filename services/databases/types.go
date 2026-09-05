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
	Scopes      []collections.Option
	Rules       []RuleView
}

type RuleView struct {
	Identifier uint
	Network    string
	Note       string
}
