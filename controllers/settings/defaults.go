package settings

import "time"

const (
	LogPrefix = "SettingsController"

	IndexPath = "/settings"

	ThemeCookie   = "sqldash-theme"
	ThemeLifetime = 8760 * time.Hour
)

var Themes = []string{"system", "light", "dark"}
