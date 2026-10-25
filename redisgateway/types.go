package redisgateway

import (
	"context"
	"crypto/tls"
	"net"
	"net/netip"
	"sync"
	"time"
)

type Binding struct {
	Username     string
	DatabaseID   uint
	CredentialID uint
	Generation   uint64
	ClientID     int64
	State        any
}

type Options struct {
	ListenAddress              string
	Listener                   net.Listener
	TLSConfig                  *tls.Config
	AllowPlaintext             bool
	MaximumConnections         int
	MaximumSessionsPerDatabase int
	MaximumHandshakes          int
	HandshakeTimeout           time.Duration
	ValidationInterval         time.Duration
	ValidationTimeout          time.Duration
	TerminationTimeout         time.Duration
	Authorize                  func(context.Context, string, netip.Addr) (*Binding, error)
	Validate                   func(context.Context, *Binding, netip.Addr) error
	Dial                       func(context.Context, *Binding) (net.Conn, error)
	Terminate                  func(context.Context, *Binding) error
	Unsafe                     func(context.Context, *Binding, error)
}

type gateway struct {
	options      Options
	mutex        sync.Mutex
	sessions     map[*session]struct{}
	databases    map[uint]int
	handshakes   chan struct{}
	callbacks    chan struct{}
	terminations chan struct{}
	enforcements chan struct{}
	failures     chan error
	listener     net.Listener
	waiting      sync.WaitGroup
}

type session struct {
	server      *gateway
	client      net.Conn
	backend     net.Conn
	binding     *Binding
	peer        netip.Addr
	databaseID  uint
	mutex       sync.Mutex
	closed      bool
	enforcement sync.Once
	done        chan struct{}
}
