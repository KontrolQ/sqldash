package auth

import (
	"strings"

	"sqldash/models"
)

func ToAccountView(record *models.Account) AccountView {
	initial := ""
	if record.Username != "" {
		initial = strings.ToUpper(record.Username[:1])
	}

	return AccountView{
		Username: record.Username,
		Email:    record.Email,
		Initial:  initial,
	}
}
