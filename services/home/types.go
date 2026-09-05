package home

type IndexContext struct {
	Title     string
	Databases []DatabaseView
}

type DatabaseView struct {
	Name      string
	Protected bool
}
