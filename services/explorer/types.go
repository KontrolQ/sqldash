package explorer

type TableView struct {
	Name   string
	IsView bool
	Rows   int64
}

type ColumnView struct {
	Name       string
	Kind       string
	NotNull    bool
	PrimaryKey bool
	Default    string
	References string
	OnColumn   string
}

type CellView struct {
	Column    string
	Text      string
	IsNull    bool
	LinkTo    string
	Truncated bool
}

type RowView struct {
	Key   string
	Cells []CellView
}

type BrowseContext struct {
	Title        string
	Database     string
	Table        string
	IsView       bool
	Tables       []TableView
	Columns      []ColumnView
	Rows         []RowView
	Total        int64
	Page         int
	Pages        int
	PageSize     int
	Sort         string
	Direction    string
	Search       string
	Editable     bool
	KeyColumn    string
	Problem      string
	Done         string
	PreviousPage int
	NextPage     int
}

type ConsoleContext struct {
	Title     string
	Database  string
	Tables    []TableView
	Statement string
	Columns   []string
	Rows      [][]string
	Affected  int64
	Duration  string
	RowsRead  int64
	Problem   string
	Ran       bool
}
