package explorer

type CellRequest struct {
	Table   string   `form:"table"`
	Columns []string `form:"column"`
	Keys    []string `form:"key"`
	Values  []string `form:"value"`
	Clears  []string `form:"clear"`
	Back    string   `form:"back"`
}

type RowRequest struct {
	Table string   `form:"table"`
	Keys  []string `form:"key"`
	Back  string   `form:"back"`
}

type ConsoleRequest struct {
	Statement string `form:"statement"`
	Snippet   uint   `form:"snippet"`
}

type SnippetRequest struct {
	Snippet   uint   `form:"snippet"`
	Name      string `form:"name"`
	Statement string `form:"statement"`
}

type InsertRequest struct {
	Table   string `form:"table"`
	Columns string `form:"columns"`
	Back    string `form:"back"`
}
