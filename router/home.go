package router

import (
	page "sqldash/pages/home"
	"sqldash/utils/urls"
)

func init() {
	urls.SetNamespace("")

	urls.Path(urls.Get, "/", page.Index, "home")
}
