package rediswire

import (
	"bytes"
	"io"
	"strconv"
)

func WriteAll(writer io.Writer, contents []byte) error {
	for len(contents) > 0 {
		written, err := writer.Write(contents)
		if err != nil {
			return err
		}
		if written < 1 {
			return io.ErrShortWrite
		}
		contents = contents[written:]
	}
	return nil
}

func WriteValue(writer io.Writer, value Value) error {
	prefix := []byte{value.Kind}
	switch value.Kind {
	case '+', '-', ':', ',', '(', '#':
		if err := WriteAll(writer, prefix); err != nil {
			return err
		}
		if bytes.ContainsAny(value.Bytes, "\r\n") {
			return InvalidFrame
		}
		if err := WriteAll(writer, value.Bytes); err != nil {
			return err
		}
		return WriteAll(writer, []byte("\r\n"))
	case '_':
		return WriteAll(writer, []byte("_\r\n"))
	case '$', '!', '=':
		length := len(value.Bytes)
		if value.Null {
			if value.Kind != '$' {
				return InvalidFrame
			}
			length = -1
		}
		prefix = strconv.AppendInt(prefix, int64(length), 10)
		prefix = append(prefix, '\r', '\n')
		if err := WriteAll(writer, prefix); err != nil {
			return err
		}
		if value.Null {
			return nil
		}
		if err := WriteAll(writer, value.Bytes); err != nil {
			return err
		}
		return WriteAll(writer, []byte("\r\n"))
	case '*', '~', '>', '%', '|':
		count := len(value.Values)
		if value.Kind == '%' || value.Kind == '|' {
			if count%2 != 0 {
				return InvalidFrame
			}
			count /= 2
		}
		if value.Null {
			if value.Kind != '*' {
				return InvalidFrame
			}
			count = -1
		}
		prefix = strconv.AppendInt(prefix, int64(count), 10)
		prefix = append(prefix, '\r', '\n')
		if err := WriteAll(writer, prefix); err != nil {
			return err
		}
		for _, child := range value.Values {
			if err := WriteValue(writer, child); err != nil {
				return err
			}
		}
		return nil
	default:
		return InvalidFrame
	}
}

func EncodeCommand(arguments ...[]byte) []byte {
	var output bytes.Buffer
	output.WriteByte('*')
	output.WriteString(strconv.Itoa(len(arguments)))
	output.WriteString("\r\n")
	for _, argument := range arguments {
		output.WriteByte('$')
		output.WriteString(strconv.Itoa(len(argument)))
		output.WriteString("\r\n")
		output.Write(argument)
		output.WriteString("\r\n")
	}
	return output.Bytes()
}
