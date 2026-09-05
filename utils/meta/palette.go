package meta

type PaletteItem struct {
	Label string
	Note  string
	URL   string
}

type PaletteGroup struct {
	Label string
	Items []PaletteItem
}
