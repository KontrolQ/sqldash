package rediswire

import "errors"

var (
	InvalidFrame   = errors.New("invalid Redis protocol frame")
	LimitExceeded  = errors.New("Redis protocol limit exceeded")
	InvalidCommand = errors.New("invalid Redis command syntax")
)
