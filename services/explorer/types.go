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

type ChoiceView struct {
	Value string
	Label string
}

type FilterView struct {
	Column    string
	Operator  string
	Value     string
	Label     string
	RemoveURL string
}

type CellView struct {
	Column     string
	Text       string
	IsNull     bool
	LinkTo     string
	LinkColumn string
	Truncated  bool
	Peek       *PeekView
}

type PeekCell struct {
	Text   string
	IsNull bool
}

type PeekView struct {
	Table   string
	Columns []string
	Cells   []PeekCell
}

type RowView struct {
	Key   string
	Cells []CellView
}

type BrowseContext struct {
	Title         string
	Database      string
	Address       string
	Table         string
	IsView        bool
	Tables        []TableView
	Columns       []ColumnView
	Rows          []RowView
	Total         int64
	Page          int
	Pages         int
	PageSize      int
	Sort          string
	Direction     string
	Search        string
	Editable      bool
	KeyColumn     string
	Filters       []FilterView
	Operators     []ChoiceView
	ColumnChoices []ChoiceView
	Carry         string
	View          string
	Duration      string
	PreviousPage  int
	NextPage      int
	FirstRow      int
	LastRow       int
	ColumnNames   string
	BrowsePath    string
	StructurePath string
}

type SnippetView struct {
	ID        uint
	Name      string
	Statement string
}

type SchemaTable struct {
	Name    string
	Columns []string
}

type ConsoleContext struct {
	Title      string
	Database   string
	Address    string
	Tables     []TableView
	BrowsePath string
	Table      string
	Schema     string
	Statement  string
	Columns    []string
	Rows       [][]string
	Affected   int64
	Duration   string
	RowsRead   int64
	Failure    string
	Ran        bool
	Snippets   []SnippetView
	SnippetID  uint
	Name       string
}

type IndexView struct {
	Name    string
	Unique  bool
	Origin  string
	Columns []string
}

type StructureContext struct {
	Title         string
	BrowsePath    string
	StructurePath string
	Database      string
	Address       string
	Table         string
	IsView        bool
	Tables        []TableView
	Columns       []ColumnView
	Indexes       []IndexView
	Triggers      []TriggerView
	Relations     []RelationView
	UsedBy        []RelationView
	Definition    string
}

type TriggerView struct {
	Name       string
	Definition string
}

type RelationView struct {
	FromTable  string
	FromColumn string
	ToTable    string
	ToColumn   string
	OnDelete   string
	OnUpdate   string
}
