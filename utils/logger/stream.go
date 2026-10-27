package logger

import (
	"bytes"
	"io"
	"sync"

	"go.uber.org/zap/zapcore"
)

const MaximumOutputLineBytes = 64 * 1024

type processStream struct {
	mutex   sync.Mutex
	prefix  string
	format  OutputFormat
	pending []byte
	level   zapcore.Level
	closed  bool
}

func ProcessStream(prefix string, format OutputFormat) io.WriteCloser {
	return &processStream{prefix: prefix, format: format, level: zapcore.InfoLevel}
}

func (self *processStream) Write(contents []byte) (int, error) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if self.closed {
		return 0, io.ErrClosedPipe
	}
	length := len(contents)
	for len(contents) > 0 {
		available := min(len(contents), MaximumOutputLineBytes-len(self.pending))
		part := contents[:available]
		if newline := bytes.IndexByte(part, '\n'); newline >= 0 {
			self.pending = append(self.pending, part[:newline]...)
			self.flush()
			contents = contents[newline+1:]
		} else {
			self.pending = append(self.pending, part...)
			contents = contents[available:]
			if len(self.pending) == MaximumOutputLineBytes {
				self.flush()
			}
		}
	}
	return length, nil
}

func (self *processStream) Close() error {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if !self.closed {
		self.flush()
		self.closed = true
		self.pending = nil
	}
	return nil
}

func (self *processStream) flush() {
	if len(self.pending) > 0 {
		level, message := processLine(self.format, string(self.pending), self.level)
		self.level = level
		emit(processLabel(level), level, self.prefix, message)
		self.pending = self.pending[:0]
	}
}
