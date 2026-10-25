package redishost

import "errors"

var (
	Unavailable       = errors.New("Redis instance is unavailable")
	IdentityMismatch  = errors.New("Redis instance identity could not be verified")
	UnsafePath        = errors.New("Redis storage path is unsafe")
	CapacityExceeded  = errors.New("Redis instance capacity is exhausted")
	PolicyUnconfirmed = errors.New("Redis access policy could not be confirmed")
	InvalidSettings   = errors.New("Redis settings are invalid")
	InvalidCredential = errors.New("Redis credential is invalid")
	StorageUnowned    = errors.New("Existing Redis storage is not owned by this database")
)
