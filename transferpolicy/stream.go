package transferpolicy

import (
	"io"
	"math"

	"sqldash/config"
)

type quotaWriter struct {
	destination io.Writer
	maximum     int64
	written     int64
}

func Writer(destination io.Writer) io.Writer {
	return &quotaWriter{destination: destination, maximum: config.Transfer.MaximumBytes}
}

func (self *quotaWriter) Write(contents []byte) (int, error) {
	if int64(len(contents)) > math.MaxInt64-self.written || self.maximum > 0 && int64(len(contents)) > self.maximum-self.written {
		return 0, SizeExceeded
	}
	written, err := self.destination.Write(contents)
	self.written += int64(written)
	return written, err
}

func Copy(destination io.Writer, source io.Reader) (int64, error) {
	return io.CopyBuffer(Writer(destination), source, make([]byte, CopyBufferBytes))
}
