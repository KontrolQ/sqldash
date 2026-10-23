package control

import (
	"context"
	"errors"
	"net"
	"sync/atomic"

	"github.com/jackc/pgx/v5"
)

var ResponseLimit = errors.New("the PostgreSQL response exceeds the session limit")

type responseConnection struct {
	net.Conn
	remaining atomic.Int64
}

func (self *responseConnection) Read(buffer []byte) (int, error) {
	remaining := self.remaining.Load()
	if remaining == 0 {
		return 0, ResponseLimit
	}
	if remaining > 0 && int64(len(buffer)) > remaining {
		buffer = buffer[:remaining]
	}
	read, readError := self.Conn.Read(buffer)
	if remaining >= 0 {
		self.remaining.Add(-int64(read))
	}
	return read, readError
}

func limitTransport(configuration *pgx.ConnConfig) {
	dial := configuration.DialFunc
	configuration.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
		connection, dialError := dial(ctx, network, address)
		if dialError != nil {
			return nil, dialError
		}
		bounded := &responseConnection{Conn: connection}
		bounded.remaining.Store(-1)
		return bounded, nil
	}
}

func LimitResponseBytes(connection *pgx.Conn, maximum int64) error {
	transport, bounded := connection.PgConn().Conn().(*responseConnection)
	if !bounded || maximum == 0 || maximum < -1 {
		return InvalidConnection
	}
	transport.remaining.Store(maximum)
	return nil
}
