package router

import (
	controller "sqldash/controllers/auth"
	page "sqldash/pages/auth"
	"sqldash/utils/urls"
)

func init() {
	urls.SetNamespace("")

	urls.Path(urls.Get, "/login", page.Login, "auth.login")
	urls.Path(urls.Post, "/login", controller.SignIn, "auth.signin")
	urls.Path(urls.Get, "/setup", page.Setup, "auth.setup")
	urls.Path(urls.Post, "/setup", controller.CreateAccount, "auth.create")
	urls.Path(urls.Post, "/logout", controller.SignOut, "auth.logout")
}
