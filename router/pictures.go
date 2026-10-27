package router

import (
	controller "sqldash/controllers/settings"
	"sqldash/utils/urls"
)

func init() {
	urls.SetNamespace("pictures")
	urls.Path(urls.Get, "/:filename", controller.ShowPicture, "show")
	urls.Path(urls.Head, "/:filename", controller.ShowPicture, "head")
}
