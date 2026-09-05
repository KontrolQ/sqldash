package router

import (
	page "sqldash/pages/insights"
	"sqldash/utils/urls"
)

func init() {
	urls.SetNamespace("")

	urls.Path(urls.Get, "/insights", page.Overview, "insights")
}
