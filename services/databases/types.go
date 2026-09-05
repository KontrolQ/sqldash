package databases

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
