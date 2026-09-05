package middleware

const (
	LogPrefix = "Middleware"

	LoginPath  = "/login"
	SetupPath  = "/setup"
	StaticPath = "/static"

	ThemeCookie = "sqldash-theme"
	ThemeSystem = "system"
	ThemeLight  = "light"
	ThemeDark   = "dark"
)

var OpenPaths = []string{LoginPath, SetupPath}

var Themes = []string{ThemeSystem, ThemeLight, ThemeDark}
