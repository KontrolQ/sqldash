package settings

import "time"

const (
	LogPrefix = "SettingsController"

	IndexPath = "/settings"

	ThemeCookie   = "sqldash-theme"
	ThemeLifetime = 8760 * time.Hour

	ProblemParameter = "problem"
	DoneParameter    = "done"
)

var Themes = []string{"system", "light", "dark"}
