package rediswire

import (
	"bufio"
	"bytes"
	"net"
	"sync"
)

type guardedConnection struct {
	net.Conn
	reader  *bufio.Reader
	limits  Limits
	mutex   sync.Mutex
	pending []byte
	failure error
}

func Guard(connection net.Conn, limits Limits) net.Conn {
	return &guardedConnection{Conn: connection, reader: bufio.NewReaderSize(connection, 4096), limits: normalizeLimits(limits)}
}

func (self *guardedConnection) Read(contents []byte) (int, error) {
	if len(contents) == 0 {
		return 0, nil
	}
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if len(self.pending) == 0 {
		if self.failure != nil {
			return 0, self.failure
		}
		value, err := ReadValue(self.reader, self.limits)
		if err != nil {
			self.failure = err
			self.Conn.Close()
			return 0, err
		}
		var encoded bytes.Buffer
		if err := WriteValue(&encoded, value); err != nil {
			self.failure = err
			self.Conn.Close()
			return 0, err
		}
		self.pending = encoded.Bytes()
	}
	copied := copy(contents, self.pending)
	self.pending = self.pending[copied:]
	if len(self.pending) == 0 {
		self.pending = nil
	}
	return copied, nil
}
