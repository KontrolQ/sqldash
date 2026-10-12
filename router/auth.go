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
	urls.Path(urls.Get, "/join", page.Join, "auth.join")
	urls.Path(urls.Post, "/join", controller.AcceptInvitation, "auth.join.accept")
	urls.Path(urls.Get, "/recover", page.Recover, "auth.recover")
	urls.Path(urls.Post, "/recover", controller.RecoverAccount, "auth.recover.accept")
}
