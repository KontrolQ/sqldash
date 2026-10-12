package settings

import (
	"sqldash/models"
	"sqldash/pictures"
)

type PictureView struct {
	ID       uint
	Source   string
	Selected bool
}

func pictureHistory(account *models.Account, history []models.AccountPicture) []PictureView {
	shown := make([]PictureView, 0, len(history))
	for _, picture := range history {
		shown = append(shown, PictureView{
			ID:       picture.ID,
			Source:   pictures.Source(picture.ProfilePicture, picture.ProfilePictureType),
			Selected: picture.ProfilePicture == account.ProfilePicture && picture.ProfilePictureType == account.ProfilePictureType,
		})
	}
	return shown
}
