package rediswire

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"
)

type decoder struct {
	reader    *bufio.Reader
	limits    Limits
	remaining int64
	elements  int
}

func normalizeLimits(limits Limits) Limits {
	if limits.MaximumBytes <= 0 {
		limits.MaximumBytes = DefaultMaximumBytes
	}
	if limits.MaximumBulkBytes <= 0 || limits.MaximumBulkBytes > limits.MaximumBytes {
		limits.MaximumBulkBytes = limits.MaximumBytes
	}
	if limits.MaximumBulkBytes > MaximumBulkBytes {
		limits.MaximumBulkBytes = MaximumBulkBytes
	}
	if limits.MaximumElements <= 0 {
		limits.MaximumElements = DefaultMaximumElements
	}
	if limits.MaximumDepth <= 0 {
		limits.MaximumDepth = DefaultMaximumDepth
	}
	if limits.MaximumLineBytes <= 0 {
		limits.MaximumLineBytes = DefaultMaximumLineBytes
	}
	if int64(limits.MaximumLineBytes) > limits.MaximumBytes {
		limits.MaximumLineBytes = int(limits.MaximumBytes)
	}
	return limits
}

func ReadValue(reader *bufio.Reader, limits Limits) (Value, error) {
	limits = normalizeLimits(limits)
	decoder := &decoder{reader: reader, limits: limits, remaining: limits.MaximumBytes, elements: limits.MaximumElements}
	return decoder.read(0)
}

func readLine(reader *bufio.Reader, maximum int) ([]byte, error) {
	var contents []byte
	for {
		fragment, err := reader.ReadSlice('\n')
		if len(contents)+len(fragment) > maximum {
			return nil, LimitExceeded
		}
		contents = append(contents, fragment...)
		if err == nil {
			break
		}
		if !errors.Is(err, bufio.ErrBufferFull) {
			return nil, err
		}
	}
	if len(contents) < 2 || contents[len(contents)-2] != '\r' {
		return nil, InvalidFrame
	}
	return contents[:len(contents)-2], nil
}

func decimal(contents []byte) (int64, error) {
	if len(contents) == 0 {
		return 0, InvalidFrame
	}
	start := 0
	if contents[0] == '-' || contents[0] == '+' {
		start = 1
	}
	if start == len(contents) {
		return 0, InvalidFrame
	}
	for _, character := range contents[start:] {
		if character < '0' || character > '9' {
			return 0, InvalidFrame
		}
	}
	value, err := strconv.ParseInt(string(contents), 10, 64)
	if err != nil {
		return 0, LimitExceeded
	}
	return value, nil
}

func (self *decoder) consume(count int64) error {
	if count < 0 || count > self.remaining {
		return LimitExceeded
	}
	self.remaining -= count
	return nil
}

func (self *decoder) read(depth int) (Value, error) {
	if depth > self.limits.MaximumDepth || self.elements < 1 {
		return Value{}, LimitExceeded
	}
	self.elements--
	if err := self.consume(1); err != nil {
		return Value{}, err
	}
	kind, err := self.reader.ReadByte()
	if err != nil {
		return Value{}, err
	}
	line, err := readLine(self.reader, self.limits.MaximumLineBytes)
	if err != nil {
		return Value{}, err
	}
	if err := self.consume(int64(len(line)) + 2); err != nil {
		return Value{}, err
	}
	value := Value{Kind: kind}
	switch kind {
	case '+', '-':
		value.Bytes = line
	case ':':
		if _, err := decimal(line); err != nil {
			return Value{}, err
		}
		value.Bytes = line
	case '(':
		start := 0
		if len(line) > 0 && (line[0] == '-' || line[0] == '+') {
			start = 1
		}
		if len(line) <= start {
			return Value{}, InvalidFrame
		}
		for _, character := range line[start:] {
			if character < '0' || character > '9' {
				return Value{}, InvalidFrame
			}
		}
		value.Bytes = line
	case ',':
		lower := strings.ToLower(string(line))
		if lower != "nan" && lower != "inf" && lower != "+inf" && lower != "-inf" {
			if _, err := strconv.ParseFloat(string(line), 64); err != nil {
				return Value{}, InvalidFrame
			}
		}
		value.Bytes = line
	case '#':
		if !bytes.Equal(line, []byte("t")) && !bytes.Equal(line, []byte("f")) {
			return Value{}, InvalidFrame
		}
		value.Bytes = line
	case '_':
		if len(line) != 0 {
			return Value{}, InvalidFrame
		}
		value.Null = true
	case '$', '!', '=':
		length, err := decimal(line)
		if err != nil {
			return Value{}, err
		}
		if length == -1 && kind == '$' {
			value.Null = true
			return value, nil
		}
		if length < 0 {
			return Value{}, InvalidFrame
		}
		if length > self.limits.MaximumBulkBytes || length > self.remaining-2 {
			return Value{}, LimitExceeded
		}
		if err := self.consume(length + 2); err != nil {
			return Value{}, err
		}
		contents := make([]byte, int(length)+2)
		if _, err := io.ReadFull(self.reader, contents); err != nil {
			return Value{}, err
		}
		if contents[len(contents)-2] != '\r' || contents[len(contents)-1] != '\n' {
			return Value{}, InvalidFrame
		}
		value.Bytes = contents[:len(contents)-2]
		if kind == '=' && (len(value.Bytes) < 4 || value.Bytes[3] != ':') {
			return Value{}, InvalidFrame
		}
	case '*', '~', '>', '%', '|':
		count, err := decimal(line)
		if err != nil {
			return Value{}, err
		}
		if count == -1 && kind == '*' {
			value.Null = true
			return value, nil
		}
		if count < 0 {
			return Value{}, InvalidFrame
		}
		if count > int64(self.elements) {
			return Value{}, LimitExceeded
		}
		if kind == '%' || kind == '|' {
			if count > int64(self.elements)/2 {
				return Value{}, LimitExceeded
			}
			count *= 2
		}
		value.Values = make([]Value, 0, int(count))
		for range int(count) {
			child, err := self.read(depth + 1)
			if err != nil {
				return Value{}, err
			}
			value.Values = append(value.Values, child)
		}
	default:
		return Value{}, InvalidFrame
	}
	return value, nil
}
