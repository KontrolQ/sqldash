package router

import (
	page "sqldash/pages/audit"
	"sqldash/utils/urls"
)

func init() {
	urls.SetNamespace("")

	urls.Path(urls.Get, "/audit", page.Log, "audit")
}
