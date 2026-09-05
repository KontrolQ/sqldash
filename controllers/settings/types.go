package settings

type DetailsRequest struct {
	Username string `form:"username"`
	Email    string `form:"email"`
}

type PasswordRequest struct {
	Current string `form:"current"`
	Wanted  string `form:"wanted"`
}

type ThemeRequest struct {
	Theme string `form:"theme"`
}
