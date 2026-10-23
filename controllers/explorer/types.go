package explorer

type CellRequest struct {
	ValueEncoding string   `form:"value_encoding"`
	Schema        string   `form:"schema"`
	States        []string `form:"state"`
	Table         string   `form:"table"`
	Columns       []string `form:"column"`
	Keys          []string `form:"key"`
	Values        []string `form:"value"`
	Clears        []string `form:"clear"`
	Back          string   `form:"back"`
}

type RowRequest struct {
	Schema string   `form:"schema"`
	Table  string   `form:"table"`
	Keys   []string `form:"key"`
	Back   string   `form:"back"`
}

type ConsoleRequest struct {
	Schema    string `form:"schema"`
	RunID     string `form:"run_id"`
	Statement string `form:"statement"`
	Snippet   uint   `form:"snippet"`
}

type SnippetRequest struct {
	Snippet   uint   `form:"snippet"`
	Name      string `form:"name"`
	Statement string `form:"statement"`
}

type InsertRequest struct {
	Schema  string   `form:"schema"`
	Names   []string `form:"column"`
	States  []string `form:"state"`
	Values  []string `form:"value"`
	Table   string   `form:"table"`
	Columns string   `form:"columns"`
	Back    string   `form:"back"`
}
