package settings

type DetailsRequest struct {
	Username    string `form:"username"`
	Email       string `form:"email"`
	DisplayName string `form:"display_name"`
}

type PasswordRequest struct {
	Current string `form:"current"`
	Wanted  string `form:"wanted"`
}

type PictureSelectionRequest struct {
	PictureID uint `form:"picture_id"`
}

type ThemeRequest struct {
	Theme string `form:"theme"`
}
