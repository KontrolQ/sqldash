package members

import "time"

const (
	LogPrefix          = "Members"
	Invitation         = "invitation"
	Recovery           = "recovery"
	InvitationLifetime = 7 * 24 * time.Hour
	RecoveryLifetime   = time.Hour
	TransferLifetime   = 7 * 24 * time.Hour
	Pending            = "pending"
)
