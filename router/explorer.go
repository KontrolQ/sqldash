package router

import (
	controller "sqldash/controllers/explorer"
	page "sqldash/pages/explorer"
	"sqldash/utils/urls"
)

func init() {
	urls.SetNamespace("")

	urls.Path(urls.Get, "/databases/:name/explore", page.Browse, "databases.explore")
	urls.Path(urls.Get, "/databases/:name/structure", page.Structure, "databases.structure")
	urls.Path(urls.Post, "/databases/:name/explore/row/add", controller.InsertRow, "databases.explore.insert")
	urls.Path(urls.Get, "/databases/:name/console", page.Console, "databases.console")
	urls.Path(urls.Post, "/databases/:name/explore/cell", controller.SaveCells, "databases.explore.cell")
	urls.Path(urls.Post, "/databases/:name/explore/row", controller.DeleteRows, "databases.explore.row")
	urls.Path(urls.Post, "/databases/:name/console", controller.RunConsole, "databases.console.run")
}
