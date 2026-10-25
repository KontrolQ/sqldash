package pggateway

import (
	"context"
	"crypto/tls"
	"net"
	"sqldash/nativetls"
	"sync"
	"time"
)

type Identity struct {
	CredentialID uint
	DatabaseID   uint
	AccountID    uint
	RoleName     string
	DatabaseName string
	SourceIP     string
	Generation   uint64
}

type Options struct {
	ListenAddress              string
	BackendAddress             string
	TLSConfig                  *tls.Config
	AllowPlaintext             bool
	Registry                   *Registry
	MaximumConnections         int
	MaximumSessions            int
	MaximumHandshakes          int
	MaximumStartupBytes        int
	MaximumHandshakeFrameBytes int
	MaximumHandshakeBytes      int
	MaximumCancelRequests      int
	MaximumSessionCancels      int
	HandshakeTimeout           time.Duration
	TLSHandshakeTimeout        time.Duration
	BackendConnectTimeout      time.Duration
	CancelTimeout              time.Duration
	ValidationInterval         time.Duration
	ValidationTimeout          time.Duration
	TerminationTimeout         time.Duration
	Authorize                  func(context.Context, string, string, net.IP) (*Identity, error)
	Validate                   func(context.Context, *Identity) error
	Terminate                  func(context.Context, *Identity, uint32) error
	Unsafe                     func(error)
}

type Registry struct {
	mutex    sync.Mutex
	sessions map[*session]struct{}
}

type TLSOptions = nativetls.Options

type TLSMaterial = nativetls.Material
