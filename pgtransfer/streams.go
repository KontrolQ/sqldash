package pgtransfer

import (
	"bytes"
	"context"
	"io"
	"math"
)

type contextReader struct {
	context context.Context
	source  io.Reader
}

func (self *contextReader) Read(destination []byte) (int, error) {
	if contextError := self.context.Err(); contextError != nil {
		return 0, contextError
	}
	return self.source.Read(destination)
}

type boundedWriter struct {
	destination  io.Writer
	maximumBytes int64
	writtenBytes int64
	exceeded     bool
	limitError   error
	cancel       context.CancelFunc
}

func (self *boundedWriter) Write(source []byte) (int, error) {
	if int64(len(source)) > math.MaxInt64-self.writtenBytes || self.maximumBytes > 0 && int64(len(source)) > self.maximumBytes-self.writtenBytes {
		self.exceeded = true
		if self.cancel != nil {
			self.cancel()
		}
		return 0, self.limitError
	}
	written, writeError := self.destination.Write(source)
	self.writtenBytes += int64(written)
	if writeError != nil && self.cancel != nil {
		self.cancel()
	}
	return written, writeError
}

type diagnosticBuffer struct {
	buffer bytes.Buffer
}

func (self *diagnosticBuffer) Write(source []byte) (int, error) {
	remaining := MaximumDiagnosticBytes - self.buffer.Len()
	if remaining > 0 {
		self.buffer.Write(source[:min(len(source), remaining)])
	}
	return len(source), nil
}

func (self *diagnosticBuffer) clear() {
	clear(self.buffer.Bytes())
	self.buffer.Reset()
}
