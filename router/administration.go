package router

import (
	controller "sqldash/controllers/administration"
	page "sqldash/pages/administration"
	"sqldash/utils/urls"
)

func init() {
	urls.SetNamespace("users")
	urls.Path(urls.Get, "/", page.Users, "index")
	urls.Path(urls.Post, "/invite", controller.IssueInvitation, "invite")
	urls.Path(urls.Post, "/:identifier/settings", controller.SaveUser, "save")
	urls.Path(urls.Post, "/:identifier/active", controller.Activate, "active")
	urls.Path(urls.Post, "/:identifier/recovery", controller.IssueRecovery, "recovery")
	urls.Path(urls.Post, "/:identifier/sessions/revoke", controller.RevokeSessions, "sessions.revoke")
	urls.Path(urls.Post, "/invitations/:identifier/cancel", controller.CancelInvitation, "invitation.cancel")
	urls.Path(urls.Post, "/invitations/:identifier/reissue", controller.ReissueInvitation, "invitation.reissue")
	urls.Path(urls.Post, "/:identifier/ownership", controller.TransferOwnership, "ownership")
	urls.SetNamespace("admin")
	urls.Path(urls.Get, "/smtp", page.SMTP, "smtp")
	urls.Path(urls.Post, "/smtp", controller.SaveSMTP, "smtp.save")
	urls.Path(urls.Post, "/smtp/clear", controller.ClearSMTP, "smtp.clear")
	urls.Path(urls.Post, "/smtp/test", controller.TestSMTP, "smtp.test")
	urls.Path(urls.Get, "/roles", page.Roles, "roles")
	urls.Path(urls.Post, "/roles", controller.SaveRole, "roles.create")
	urls.Path(urls.Post, "/roles/reorder", controller.ReorderRoles, "roles.reorder")
	urls.Path(urls.Post, "/roles/:identifier", controller.SaveRole, "roles.save")
	urls.Path(urls.Post, "/roles/:identifier/delete", controller.DeleteRole, "roles.delete")
	urls.Path(urls.Post, "/roles/:identifier/users", controller.ChangeRoleMembers, "roles.users")
	urls.SetNamespace("")
	urls.Path(urls.Get, "/databases/:name/sharing", page.Sharing, "databases.sharing")
	urls.Path(urls.Post, "/databases/:name/sharing", controller.ShareDatabase, "databases.share")
	urls.Path(urls.Post, "/ownership/:identifier/respond", controller.RespondOwnership, "ownership.respond")
}
