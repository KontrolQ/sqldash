package auth

const (
	CountFailedLog  = "Failed to count accounts: %v"
	LookupFailedLog = "Failed to look up an account: %v"
	CreateFailedLog = "Failed to create the account: %v"
)

const (
	AccountUnavailable = "The account could not be read."
	CredentialsWrong   = "That username and password do not match."
	AlreadySetUp       = "An account already exists."
	DetailsMissing     = "Every field is required."
)
