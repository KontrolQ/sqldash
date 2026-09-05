package tokens

const (
	ListFailedLog   = "Failed to list tokens for %s: %v"
	MintFailedLog   = "Failed to mint a token for %s: %v"
	RevokeFailedLog = "Failed to revoke token %s: %v"
)

const (
	ListUnavailable = "The tokens could not be read."
	MintRefused     = "The token could not be created."
	TokenMissing    = "No such token."
	RevokeRefused   = "The token could not be revoked."
	ScopeUnknown    = "That is not a scope a token can have."
)
