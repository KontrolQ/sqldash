package router

import (
	controller "sqldash/controllers/databases"
	tokencontroller "sqldash/controllers/tokens"
	page "sqldash/pages/databases"
	insightpage "sqldash/pages/insights"
	redispage "sqldash/pages/redis"
	"sqldash/utils/urls"
)

func init() {
	urls.SetNamespace("")

	urls.Path(urls.Get, "/", page.Index, "databases")
	urls.Path(urls.Post, "/databases", controller.Create, "databases.create")
	urls.Path(urls.Get, "/databases/:name", page.Show, "databases.show")
	urls.Path(urls.Get, "/databases/:name/insights", redisPage(insightpage.ForDatabase, redispage.Insights), "databases.insights")
	urls.Path(urls.Get, "/databases/:name/queries", redisPage(insightpage.QueriesForDatabase, redispage.Commands), "databases.queries")
	urls.Path(urls.Post, "/databases/:name/settings", controller.SaveSettings, "databases.settings")
	urls.Path(urls.Post, "/databases/import", controller.Import, "databases.import")
	urls.Path(urls.Get, "/databases/:name/export", controller.Export, "databases.export")
	urls.Path(urls.Get, "/databases/:name/export/file", controller.ExportFile, "databases.export.file")
	urls.Path(urls.Post, "/databases/:name/fork", controller.Fork, "databases.fork")
	urls.Path(urls.Post, "/databases/:name/rules", controller.AllowNetwork, "databases.rules.add")
	urls.Path(urls.Post, "/databases/:name/rules/:rule/remove", controller.RemoveNetwork, "databases.rules.remove")
	urls.Path(urls.Post, "/databases/:name/delete", controller.Delete, "databases.delete")
	urls.Path(urls.Post, "/databases/:name/tokens", tokencontroller.Mint, "databases.tokens.mint")
	urls.Path(urls.Post, "/databases/:name/tokens/:identifier/revoke", tokencontroller.Revoke, "databases.tokens.revoke")
}
