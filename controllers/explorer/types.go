package explorer

type CellRequest struct {
	Table  string `form:"table"`
	Column string `form:"column"`
	Key    string `form:"key"`
	Value  string `form:"value"`
	Clear  string `form:"clear"`
	Back   string `form:"back"`
}

type RowRequest struct {
	Table string `form:"table"`
	Key   string `form:"key"`
	Back  string `form:"back"`
}

type ConsoleRequest struct {
	Statement string `form:"statement"`
}

type InsertRequest struct {
	Table   string `form:"table"`
	Columns string `form:"columns"`
	Back    string `form:"back"`
}
