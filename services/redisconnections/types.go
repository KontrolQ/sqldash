package redisconnections

import (
	"time"

	"sqldash/models"
)

type Creation struct {
	Label     string
	Scope     models.TokenScope
	ExpiresAt *time.Time
	Subscribe bool
	Publish   bool
}

type Issued struct {
	CredentialID uint              `json:"credentialID"`
	Username     string            `json:"username"`
	Password     string            `json:"password"`
	URL          string            `json:"url"`
	Scope        models.TokenScope `json:"scope"`
	ExpiresAt    *time.Time        `json:"expiresAt"`
}

type pendingPassword struct {
	DatabaseID         uint
	CredentialID       uint
	AccountID          uint
	Generation         uint64
	InstanceIdentity   string
	InstanceGeneration uint64
	Scope              models.TokenScope
	Subscribe          bool
	Publish            bool
	CreatedAt          time.Time
	Password           string
}
