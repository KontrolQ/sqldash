package administration

import (
	"github.com/gofiber/fiber/v3"
	"sqldash/models"
	"sqldash/services/mailing"
	"sqldash/services/members"
	"sqldash/utils/meta"
)

type SMTPRequest struct {
	Host      string `form:"host"`
	Port      int    `form:"port"`
	Security  string `form:"security"`
	Username  string `form:"username"`
	Password  string `form:"password"`
	FromEmail string `form:"from_email"`
	FromName  string `form:"from_name"`
}

func SaveSMTP(context fiber.Ctx) error {
	asked, parseError := meta.Body[SMTPRequest](context)
	if parseError != nil {
		return complete(context, SMTPSaved, members.Invalid, SMTPPath)
	}
	held := models.SMTPSettings{Host: asked.Host, Port: asked.Port, Security: asked.Security, Username: asked.Username, FromEmail: asked.FromEmail, FromName: asked.FromName}
	return complete(context, SMTPSaved, mailing.Save(held, asked.Password), SMTPPath)
}
func ClearSMTP(context fiber.Ctx) error {
	return complete(context, SMTPCleared, mailing.Clear(), SMTPPath)
}
func TestSMTP(context fiber.Ctx) error {
	sent, sendError := mailing.Send(context.FormValue("recipient"), "sqldash SMTP test", "Your SMTP configuration is working.")
	if sendError == nil && !sent {
		sendError = mailing.SettingsInvalid
	}
	return complete(context, "smtp.test", sendError, SMTPPath)
}
