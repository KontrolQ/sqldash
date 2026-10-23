package pggateway

import (
	"bufio"
	"context"
	"crypto/tls"
	"io"
	"net"
	"time"
)

type bufferedConnection struct {
	net.Conn
	reader *bufio.Reader
}

func (self *bufferedConnection) Read(contents []byte) (int, error) {
	return self.reader.Read(contents)
}

func (self *gateway) negotiate(connection net.Conn) (net.Conn, startupPacket, error) {
	deadline := time.Now().Add(self.options.HandshakeTimeout)
	if deadlineError := connection.SetDeadline(deadline); deadlineError != nil {
		return connection, startupPacket{}, deadlineError
	}
	reader := bufio.NewReader(connection)
	initial, peekError := reader.Peek(1)
	if peekError != nil {
		return connection, startupPacket{}, peekError
	}
	if initial[0] == 0x16 {
		secured, handshakeError := self.secure(connection, reader, deadline, true)
		if handshakeError != nil {
			return secured, startupPacket{}, handshakeError
		}
		packet, readError := self.readInitialPacket(secured, connection, deadline)
		if readError == nil && (packet.code == sslRequestCode || packet.code == gssRequestCode) {
			readError = ErrInvalidStartup
		}
		return secured, packet, readError
	}
	gssRefused := false
	for {
		packet, readError := self.readInitialPacket(reader, connection, deadline)
		if readError != nil {
			return connection, startupPacket{}, readError
		}
		switch packet.code {
		case gssRequestCode:
			if len(packet.contents) != 8 || gssRefused {
				return connection, startupPacket{}, ErrInvalidStartup
			}
			gssRefused = true
			if writeError := writeAll(connection, []byte{'N'}); writeError != nil {
				return connection, startupPacket{}, writeError
			}
		case sslRequestCode:
			if len(packet.contents) != 8 {
				return connection, startupPacket{}, ErrInvalidStartup
			}
			if writeError := writeAll(connection, []byte{'S'}); writeError != nil {
				return connection, startupPacket{}, writeError
			}
			secured, handshakeError := self.secure(connection, reader, deadline, false)
			if handshakeError != nil {
				return secured, startupPacket{}, handshakeError
			}
			encryptedPacket, encryptedError := self.readInitialPacket(secured, connection, deadline)
			if encryptedError == nil && (encryptedPacket.code == sslRequestCode || encryptedPacket.code == gssRequestCode) {
				encryptedError = ErrInvalidStartup
			}
			return secured, encryptedPacket, encryptedError
		default:
			if !self.options.AllowPlaintext {
				return connection, packet, ErrTLSRequired
			}
			return &bufferedConnection{Conn: connection, reader: reader}, packet, nil
		}
	}
}

func (self *gateway) readInitialPacket(reader io.Reader, connection net.Conn, deadline time.Time) (startupPacket, error) {
	packetDeadline := deadline
	packet, packetError := readStartup(reader, self.options.MaximumStartupBytes, func(code uint32) error {
		if code != cancelRequestCode {
			return nil
		}
		cancelDeadline := time.Now().Add(self.options.CancelTimeout)
		if cancelDeadline.Before(packetDeadline) {
			packetDeadline = cancelDeadline
		}
		return connection.SetDeadline(packetDeadline)
	})
	packet.deadline = packetDeadline
	return packet, packetError
}

func (self *gateway) secure(connection net.Conn, reader *bufio.Reader, deadline time.Time, direct bool) (net.Conn, error) {
	secured := tls.Server(&bufferedConnection{Conn: connection, reader: reader}, self.options.TLSConfig)
	handshakeDeadline := time.Now().Add(self.options.TLSHandshakeTimeout)
	if deadline.Before(handshakeDeadline) {
		handshakeDeadline = deadline
	}
	ctx, cancel := context.WithDeadline(self.context, handshakeDeadline)
	defer cancel()
	if handshakeError := secured.HandshakeContext(ctx); handshakeError != nil {
		return secured, handshakeError
	}
	if direct && secured.ConnectionState().NegotiatedProtocol != postgresqlApplicationProtocol {
		return secured, ErrInvalidStartup
	}
	return secured, nil
}
