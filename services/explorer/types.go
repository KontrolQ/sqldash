package explorer

type TableView struct {
	Schema string
	Kind   string
	Query  string
	Name   string
	IsView bool
	Rows   int64
}

type ColumnView struct {
	Generated       bool
	Identity        bool
	ReadOnly        bool
	PrimaryPosition int
	ReferenceSchema string
	Name            string
	Kind            string
	NotNull         bool
	PrimaryKey      bool
	Default         string
	References      string
	OnColumn        string
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
	ReadOnly   bool
	Value      string
	LinkQuery  string
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
	ReadOnlySchema bool
	Unavailable    string
	GridEditor     string
	GridData       string
	Engine         string
	Schema         string
	Schemas        []ChoiceView
	RelationQuery  string
	Insertable     bool
	Title          string
	Database       string
	Address        string
	Table          string
	IsView         bool
	Tables         []TableView
	Columns        []ColumnView
	Rows           []RowView
	Total          int64
	Page           int
	Pages          int
	PageSize       int
	Sort           string
	Direction      string
	Search         string
	Editable       bool
	KeyColumn      string
	Filters        []FilterView
	Operators      []ChoiceView
	ColumnChoices  []ChoiceView
	Carry          string
	CarryNoSearch  string
	View           string
	Duration       string
	PreviousPage   int
	NextPage       int
	FirstRow       int
	LastRow        int
	ColumnNames    string
	BrowsePath     string
	StructurePath  string
}

type SnippetView struct {
	ID        uint
	Name      string
	Statement string
}

type SchemaTable struct {
	Schema  string
	Kind    string
	Name    string
	Columns []string
}

type ConsoleContext struct {
	Engine            string
	CurrentSchema     string
	Schemas           []ChoiceView
	Results           []ConsoleResult
	RowsReadAvailable bool
	CancelPath        string
	RunID             string
	Title             string
	Database          string
	Address           string
	Tables            []TableView
	BrowsePath        string
	Table             string
	Schema            string
	Statement         string
	Columns           []string
	Rows              [][]string
	Affected          int64
	Duration          string
	RowsRead          int64
	Failure           string
	Ran               bool
	Snippets          []SnippetView
	SnippetID         uint
	Name              string
}

type IndexView struct {
	Name    string
	Unique  bool
	Origin  string
	Columns []string
}

type StructureContext struct {
	ReadOnlySchema   bool
	Unavailable      string
	MutableSchemas   []ChoiceView
	Engine           string
	Schema           string
	Schemas          []ChoiceView
	RelationQuery    string
	Constraints      []ConstraintView
	Functions        []TriggerView
	Sequences        []TriggerView
	CanManageObjects bool
	ObjectPath       string
	Kinds            []ChoiceView
	Title            string
	BrowsePath       string
	StructurePath    string
	Database         string
	Address          string
	Table            string
	IsView           bool
	Tables           []TableView
	Columns          []ColumnView
	Indexes          []IndexView
	Triggers         []TriggerView
	Relations        []RelationView
	UsedBy           []RelationView
	Definition       string
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
