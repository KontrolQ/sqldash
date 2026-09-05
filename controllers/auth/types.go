package auth

type SignInRequest struct {
	Username string `form:"username"`
	Password string `form:"password"`
}

type SetupRequest struct {
	Username string `form:"username"`
	Email    string `form:"email"`
	Password string `form:"password"`
}
