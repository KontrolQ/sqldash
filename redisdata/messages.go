package redisdata

import "errors"

var (
	InvalidInput   = errors.New("invalid Redis data input")
	InvalidScope   = errors.New("Redis data scope is required")
	InvalidVersion = errors.New("invalid Redis value reference")
	StaleValue     = errors.New("Redis value changed or expired; inspect it again")
	TargetExists   = errors.New("Redis target already exists")
	LimitExceeded  = errors.New("Redis data exceeds the bounded view")
	Unsupported    = errors.New("unsupported Redis data operation")
)
