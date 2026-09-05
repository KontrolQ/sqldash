package router

import (
	controller "sqldash/controllers/settings"
	page "sqldash/pages/settings"
	"sqldash/utils/urls"
)

func init() {
	urls.SetNamespace("settings")

	urls.Path(urls.Get, "/", page.Index, "index")
	urls.Path(urls.Post, "/details", controller.SaveDetails, "details")
	urls.Path(urls.Post, "/password", controller.ChangePassword, "password")
	urls.Path(urls.Post, "/theme", controller.ChooseTheme, "theme")
}
