package secretstore

import "errors"

var (
	ErrNotFound      = errors.New("secret not found")
	ErrInvalidName   = errors.New("invalid secret namespace or name")
	ErrUnsafePath    = errors.New("secret storage contains an unsafe path or permissions")
	ErrMissingKey    = errors.New("secret storage key is missing while protected data exists")
	ErrInvalidKey    = errors.New("secret storage key is invalid")
	ErrCorruptSecret = errors.New("protected secret could not be authenticated")
)
