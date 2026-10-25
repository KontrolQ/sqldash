package redisgateway

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"net"
	"strconv"
	"time"

	"sqldash/models"
	"sqldash/rediswire"
)

type bufferedConnection struct {
	net.Conn
	reader *bufio.Reader
}

func (self *bufferedConnection) Read(contents []byte) (int, error) { return self.reader.Read(contents) }

func authenticationUsername(arguments [][]byte, required bool) (string, error) {
	if len(arguments) == 0 {
		return "", AuthenticationFailed
	}
	if bytes.EqualFold(arguments[0], []byte("AUTH")) {
		if len(arguments) != 3 {
			return "", AuthenticationFailed
		}
		return string(arguments[1]), nil
	}
	if !bytes.EqualFold(arguments[0], []byte("HELLO")) {
		return "", AuthenticationFailed
	}
	if len(arguments) > 1 && !bytes.Equal(arguments[1], []byte("2")) && !bytes.Equal(arguments[1], []byte("3")) {
		return "", AuthenticationFailed
	}
	var username string
	for index := 2; index < len(arguments); {
		switch {
		case bytes.EqualFold(arguments[index], []byte("AUTH")):
			if username != "" || index+2 >= len(arguments) {
				return "", AuthenticationFailed
			}
			username = string(arguments[index+1])
			index += 3
		case bytes.EqualFold(arguments[index], []byte("SETNAME")):
			if index+1 >= len(arguments) {
				return "", AuthenticationFailed
			}
			index += 2
		default:
			return "", AuthenticationFailed
		}
	}
	if required && username == "" {
		return "", AuthenticationFailed
	}
	return username, nil
}

func publicUsername(username string) bool {
	return models.PublicRedisUsername(username)
}

func clearArguments(arguments [][]byte) {
	for _, argument := range arguments {
		clear(argument)
	}
}

func (self *session) authenticate(parent context.Context) (net.Conn, *bufio.Reader, *bufio.Reader, error) {
	options := self.server.options
	ctx, cancel := context.WithTimeout(parent, options.HandshakeTimeout)
	defer cancel()
	deadline, _ := ctx.Deadline()
	_ = self.client.SetDeadline(deadline)
	client := self.client
	reader := bufio.NewReaderSize(client, maximumAuthenticationBytes)
	prefix, err := reader.Peek(1)
	if err != nil {
		return nil, nil, nil, AuthenticationFailed
	}
	if prefix[0] == 0x16 {
		if options.TLSConfig == nil {
			return nil, nil, nil, AuthenticationFailed
		}
		secured := tls.Server(&bufferedConnection{Conn: client, reader: reader}, options.TLSConfig)
		if err := secured.HandshakeContext(ctx); err != nil {
			return nil, nil, nil, AuthenticationFailed
		}
		client = secured
		reader = bufio.NewReaderSize(client, maximumAuthenticationBytes)
	} else if !options.AllowPlaintext {
		_ = rediswire.WriteAll(client, []byte(tlsRequired))
		return nil, nil, nil, AuthenticationFailed
	}
	var arguments [][]byte
	var username string
	for attempt := 0; attempt < maximumAuthenticationCommands; attempt++ {
		arguments, err = rediswire.ReadCommand(reader, rediswire.Limits{MaximumBytes: maximumAuthenticationBytes, MaximumBulkBytes: maximumAuthenticationBytes, MaximumElements: 64, MaximumLineBytes: maximumAuthenticationBytes})
		if err != nil {
			return nil, nil, nil, AuthenticationFailed
		}
		username, err = authenticationUsername(arguments, true)
		if err == nil {
			break
		}
		clearArguments(arguments)
		arguments = nil
		if err := rediswire.WriteAll(client, []byte(authenticationRequired)); err != nil {
			return nil, nil, nil, AuthenticationFailed
		}
	}
	if len(arguments) == 0 || !publicUsername(username) {
		clearArguments(arguments)
		return nil, nil, nil, AuthenticationFailed
	}
	defer clearArguments(arguments)
	binding, err := self.authorize(ctx, username)
	if err != nil || binding == nil || binding.Username != username || binding.DatabaseID == 0 || binding.CredentialID == 0 || binding.Generation == 0 {
		_ = rediswire.WriteAll(client, []byte(authenticationDenied))
		return nil, nil, nil, AccessDenied
	}
	self.binding = binding
	if !self.server.reserve(self, binding.DatabaseID) {
		_ = rediswire.WriteAll(client, []byte(connectionLimit))
		return nil, nil, nil, AccessDenied
	}
	backend, err := self.dial(ctx)
	if err != nil || backend == nil {
		_ = rediswire.WriteAll(client, []byte(backendUnavailable))
		return nil, nil, nil, BackendUnavailable
	}
	if !self.setBackend(backend) {
		return nil, nil, nil, BackendUnavailable
	}
	_ = backend.SetDeadline(deadline)
	encoded := rediswire.EncodeCommand(arguments...)
	err = rediswire.WriteAll(backend, encoded)
	clear(encoded)
	if err != nil {
		return nil, nil, nil, BackendUnavailable
	}
	upstream := bufio.NewReaderSize(backend, maximumAuthenticationBytes)
	authenticated, err := rediswire.ReadValue(upstream, rediswire.Limits{MaximumBytes: maximumAuthenticationBytes, MaximumElements: 2048})
	if err != nil || authenticated.Kind == '-' || authenticated.Kind == '!' {
		_ = rediswire.WriteAll(client, []byte(authenticationDenied))
		return nil, nil, nil, AuthenticationFailed
	}
	if bytes.EqualFold(arguments[0], []byte("AUTH")) && (authenticated.Kind != '+' || !bytes.Equal(authenticated.Bytes, []byte("OK"))) {
		return nil, nil, nil, AuthenticationFailed
	}
	if bytes.EqualFold(arguments[0], []byte("HELLO")) && authenticated.Kind != '%' && authenticated.Kind != '*' {
		return nil, nil, nil, AuthenticationFailed
	}
	if err := rediswire.WriteAll(backend, rediswire.EncodeCommand([]byte("CLIENT"), []byte("ID"))); err != nil {
		return nil, nil, nil, BackendUnavailable
	}
	identity, err := rediswire.ReadValue(upstream, rediswire.Limits{MaximumBytes: 256})
	if err != nil || identity.Kind != ':' {
		return nil, nil, nil, BackendUnavailable
	}
	clientID, err := strconv.ParseInt(string(identity.Bytes), 10, 64)
	if err != nil || clientID < 1 {
		return nil, nil, nil, BackendUnavailable
	}
	binding.ClientID = clientID
	if err := self.validate(ctx); err != nil {
		self.enforce(parent)
		return nil, nil, nil, AccessDenied
	}
	if err := rediswire.WriteValue(client, authenticated); err != nil {
		return nil, nil, nil, AuthenticationFailed
	}
	_ = backend.SetDeadline(time.Time{})
	_ = client.SetDeadline(time.Time{})
	return client, reader, upstream, nil
}
