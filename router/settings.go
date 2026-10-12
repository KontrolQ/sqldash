package router

import (
	controller "sqldash/controllers/settings"
	"sqldash/middleware"
	page "sqldash/pages/settings"
	"sqldash/utils/urls"
)

func init() {
	urls.SetNamespace("settings")

	urls.Path(urls.Get, "/", page.Index, "index")
	urls.Path(urls.Post, "/details", controller.SaveDetails, "details")
	urls.Path(urls.Post, "/picture", middleware.ImageUpload(controller.PictureField, controller.IndexPath, controller.SavePicture), "picture")
	urls.Path(urls.Post, "/picture/restore", controller.RestorePicture, "picture.restore")
	urls.Path(urls.Post, "/picture/gravatar", controller.UseGravatar, "picture.gravatar")
	urls.Path(urls.Post, "/picture/clear", controller.ClearPicture, "picture.clear")
	urls.Path(urls.Post, "/picture/delete", controller.DeletePicture, "picture.delete")
	urls.Path(urls.Post, "/password", controller.ChangePassword, "password")
	urls.Path(urls.Post, "/theme", controller.ChooseTheme, "theme")
}
