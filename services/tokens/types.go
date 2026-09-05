package tokens

import "time"

type TokenView struct {
	Identifier string
	Label      string
	Prefix     string
	Scope      string
	Revoked    bool
	Expired    bool
	CreatedAt  time.Time
	LastUsedAt *time.Time
}
