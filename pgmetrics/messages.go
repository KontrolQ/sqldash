package pgmetrics

import "errors"

var IdentityMismatch = errors.New("PostgreSQL statistics identity does not match the registration")

const (
	SamplingFailed           = "Statistics sampling failed for database %d"
	RegistrationsUnavailable = "Statistics registrations could not be read"
)
