package router

import (
	controller "sqldash/controllers/explorer"
	page "sqldash/pages/explorer"
	redispage "sqldash/pages/redis"
	"sqldash/utils/urls"
)

func init() {
	urls.SetNamespace("")

	urls.Path(urls.Get, "/databases/:name/explore", redisPage(page.Browse, redispage.Data), "databases.explore")
	urls.Path(urls.Get, "/databases/:name/structure", page.Structure, "databases.structure")
	urls.Path(urls.Post, "/databases/:name/explore/row/add", controller.InsertRow, "databases.explore.insert")
	urls.Path(urls.Get, "/databases/:name/console", redisPage(page.Console, redispage.Console), "databases.console")
	urls.Path(urls.Post, "/databases/:name/explore/cell", controller.SaveCells, "databases.explore.cell")
	urls.Path(urls.Post, "/databases/:name/explore/row", controller.DeleteRows, "databases.explore.row")
	urls.Path(urls.Post, "/databases/:name/console", controller.RunConsole, "databases.console.run")
	urls.Path(urls.Post, "/databases/:name/console/cancel", controller.CancelConsole, "databases.console.cancel")
	urls.Path(urls.Post, "/databases/:name/structure/object", controller.ManageObject, "databases.structure.object")
	urls.Path(urls.Post, "/databases/:name/console/snippets", controller.SaveSnippet, "databases.console.snippet.save")
	urls.Path(urls.Post, "/databases/:name/console/snippets/remove", controller.RemoveSnippet, "databases.console.snippet.remove")
}
