package auth

import (
	"strings"
	"unicode/utf8"

	"sqldash/models"
	"sqldash/pictures"
)

func ToAccountView(record *models.Account) AccountView {
	displayName := strings.TrimSpace(record.DisplayName)
	if displayName == "" {
		displayName = record.Username
	}
	initial := ""
	if displayName != "" {
		first, _ := utf8.DecodeRuneInString(displayName)
		initial = strings.ToUpper(string(first))
	}

	return AccountView{
		Username:       record.Username,
		DisplayName:    displayName,
		Email:          record.Email,
		Initial:        initial,
		ProfilePicture: pictures.Source(record.ProfilePicture, record.ProfilePictureType),
	}
}
