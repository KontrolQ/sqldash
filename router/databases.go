package router

import (
	controller "sqldash/controllers/databases"
	page "sqldash/pages/databases"
	"sqldash/utils/urls"
)

func init() {
	urls.SetNamespace("")

	urls.Path(urls.Get, "/", page.Index, "databases")
	urls.Path(urls.Post, "/databases", controller.Create, "databases.create")
	urls.Path(urls.Post, "/databases/:name/delete", controller.Delete, "databases.delete")
}
