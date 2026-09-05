package middleware

import "sqldash/utils/meta"

const (
	LogPrefix = "Middleware"

	LoginPath  = "/login"
	SetupPath  = "/setup"
	StaticPath = "/static"

	ThemeCookie = "sqldash-theme"
	ThemeSystem = "system"
	ThemeLight  = "light"
	ThemeDark   = "dark"

	DatabasePath = "/databases/"
	ExplorePath  = "/explore"
	ConsolePath  = "/console"
	InsightsPath = "/insights"
	TableQuery   = "?table="

	PaletteGoLabel        = "Go to"
	PaletteDatabasesLabel = "Databases"
	PaletteTablesLabel    = "Tables"
	PaletteInsideLabel    = "Inside %s"
	PaletteDatabaseNote   = "Database"
	PaletteTableNote      = "Table"
	PaletteViewNote       = "View"
	PaletteOverviewLabel  = "Overview"
	PaletteDataLabel      = "Data"
	PaletteConsoleLabel   = "Console"
	PaletteInsightsLabel  = "Insights"
)

var PaletteDestinations = []meta.PaletteItem{
	{Label: "Databases", URL: "/"},
	{Label: "Insights", URL: "/insights"},
	{Label: "Settings", URL: "/settings"},
}

var OpenPaths = []string{LoginPath, SetupPath}

var Themes = []string{ThemeSystem, ThemeLight, ThemeDark}
