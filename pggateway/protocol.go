package pggateway

import (
	"encoding/binary"
	"io"
	"time"
)

type startupPacket struct {
	contents []byte
	code     uint32
	deadline time.Time
}

type protocolFrame struct {
	kind     byte
	contents []byte
}

func readStartup(reader io.Reader, maximumBytes int, onHeader func(uint32) error) (startupPacket, error) {
	header := make([]byte, 8)
	if _, readError := io.ReadFull(reader, header); readError != nil {
		return startupPacket{}, readError
	}
	length := binary.BigEndian.Uint32(header)
	if length < 8 || uint64(length) > uint64(maximumBytes) {
		return startupPacket{}, ErrInvalidStartup
	}
	code := binary.BigEndian.Uint32(header[4:])
	if ((code == sslRequestCode || code == gssRequestCode) && length != 8) || (code == cancelRequestCode && (length < 12+minimumCancelKeyBytes || length > maximumCancelPacketBytes)) {
		return startupPacket{}, ErrInvalidStartup
	}
	if onHeader != nil {
		if headerError := onHeader(code); headerError != nil {
			return startupPacket{}, headerError
		}
	}
	contents := make([]byte, int(length))
	copy(contents, header)
	if _, readError := io.ReadFull(reader, contents[8:]); readError != nil {
		return startupPacket{}, readError
	}
	return startupPacket{contents: contents, code: code}, nil
}

func readFrame(reader io.Reader, maximumBytes int, remainingBytes *int) (protocolFrame, error) {
	header := make([]byte, 5)
	if _, readError := io.ReadFull(reader, header); readError != nil {
		return protocolFrame{}, readError
	}
	length := binary.BigEndian.Uint32(header[1:])
	if length < 4 || uint64(length)+1 > uint64(maximumBytes) || uint64(length)+1 > uint64(*remainingBytes) {
		return protocolFrame{}, ErrAuthentication
	}
	*remainingBytes -= int(length) + 1
	contents := make([]byte, int(length)+1)
	copy(contents, header)
	if _, readError := io.ReadFull(reader, contents[5:]); readError != nil {
		return protocolFrame{}, readError
	}
	return protocolFrame{kind: header[0], contents: contents}, nil
}

func writeAll(writer io.Writer, contents []byte) error {
	for len(contents) > 0 {
		written, writeError := writer.Write(contents)
		if writeError != nil {
			return writeError
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		contents = contents[written:]
	}
	return nil
}

func writeProtocolError(writer io.Writer, code, message string) {
	contents := []byte{'E', 0, 0, 0, 0}
	for _, field := range []struct {
		kind byte
		text string
	}{{'S', "FATAL"}, {'V', "FATAL"}, {'C', code}, {'M', message}} {
		contents = append(contents, field.kind)
		contents = append(contents, field.text...)
		contents = append(contents, 0)
	}
	contents = append(contents, 0)
	binary.BigEndian.PutUint32(contents[1:5], uint32(len(contents)-1))
	_ = writeAll(writer, contents)
}
