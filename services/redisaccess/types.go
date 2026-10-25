package redisaccess

import (
	"net/netip"
	"sqldash/models"
)

type authority struct {
	AccountID          uint
	InstanceIdentity   string
	InstanceGeneration uint64
	RunID              string
	Scope              models.TokenScope
	Subscribe          bool
	Publish            bool
	Peer               netip.Addr
}
