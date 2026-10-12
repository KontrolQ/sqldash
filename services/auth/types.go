package auth

type GateContext struct {
	Title string
}

type AccountView struct {
	Username       string
	DisplayName    string
	Email          string
	Initial        string
	ProfilePicture string
}
