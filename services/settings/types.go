package settings

type IndexContext struct {
	Title                   string
	Username                string
	DisplayName             string
	Email                   string
	ProfilePicture          string
	PictureUploadLimit      string
	MaximumImageUploadBytes int
	PictureHistory          []PictureView
}
