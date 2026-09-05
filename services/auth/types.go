package auth

type GateContext struct {
	Title   string
	Problem string
}

type AccountView struct {
	Username string
	Email    string
	Initial  string
}
