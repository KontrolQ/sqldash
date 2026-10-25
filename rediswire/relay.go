package rediswire

import (
	"bufio"
	"bytes"
	"io"
	"strconv"
)

func normalizeRelay(options RelayOptions) RelayOptions {
	if options.MaximumBulkBytes <= 0 || options.MaximumBulkBytes > MaximumBulkBytes {
		options.MaximumBulkBytes = MaximumBulkBytes
	}
	if options.MaximumArguments <= 0 || options.MaximumArguments > MaximumArguments {
		options.MaximumArguments = MaximumArguments
	}
	if options.MaximumHeaderBytes <= 0 || options.MaximumHeaderBytes > MaximumMetadataBytes {
		options.MaximumHeaderBytes = MaximumMetadataBytes
	}
	return options
}

func readArgument(reader *bufio.Reader, remaining int) ([]byte, error) {
	if remaining < 4 {
		return nil, LimitExceeded
	}
	prefix, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	line, err := readLine(reader, remaining)
	if err != nil {
		return nil, err
	}
	if prefix == '+' {
		return line, nil
	}
	if prefix != '$' {
		return nil, InvalidFrame
	}
	length, err := decimal(line)
	if err != nil {
		return nil, err
	}
	if length < 0 {
		return nil, InvalidFrame
	}
	if length > int64(remaining-len(line)-5) {
		return nil, LimitExceeded
	}
	contents := make([]byte, int(length)+2)
	if _, err := io.ReadFull(reader, contents); err != nil {
		return nil, err
	}
	if contents[len(contents)-2] != '\r' || contents[len(contents)-1] != '\n' {
		return nil, InvalidFrame
	}
	return contents[:len(contents)-2], nil
}

func inspectedCommand(arguments [][]byte, writer io.Writer, inspect func([][]byte) error) error {
	defer func() {
		for _, argument := range arguments {
			clear(argument)
		}
	}()
	if inspect != nil {
		if err := inspect(arguments); err != nil {
			return err
		}
	}
	encoded := EncodeCommand(arguments...)
	defer clear(encoded)
	return WriteAll(writer, encoded)
}

func RelayCommand(reader *bufio.Reader, writer io.Writer, supplied RelayOptions) error {
	options := normalizeRelay(supplied)
	prefix, err := reader.Peek(1)
	if err != nil {
		return err
	}
	if prefix[0] != '*' {
		arguments, err := ReadCommand(reader, Limits{MaximumBytes: int64(options.MaximumHeaderBytes), MaximumElements: options.MaximumArguments, MaximumLineBytes: options.MaximumHeaderBytes})
		if err != nil {
			return err
		}
		return inspectedCommand(arguments, writer, options.Inspect)
	}
	_, _ = reader.ReadByte()
	line, err := readLine(reader, options.MaximumHeaderBytes)
	if err != nil {
		return err
	}
	count, err := decimal(line)
	if err != nil {
		return err
	}
	if count < 1 {
		return InvalidFrame
	}
	if count > int64(options.MaximumArguments) {
		return LimitExceeded
	}
	name, err := readArgument(reader, 512)
	if err != nil || len(name) == 0 || len(name) > 256 {
		if err != nil {
			return err
		}
		return InvalidCommand
	}
	protected := bytes.EqualFold(name, []byte("AUTH")) || bytes.EqualFold(name, []byte("HELLO")) || bytes.EqualFold(name, []byte("SELECT"))
	if protected {
		if count > 64 {
			return LimitExceeded
		}
		remaining := options.MaximumHeaderBytes - len(name) - len(line) - 16
		arguments := make([][]byte, 0, int(count))
		arguments = append(arguments, name)
		for index := int64(1); index < count; index++ {
			argument, err := readArgument(reader, remaining)
			if err != nil {
				for _, argument := range arguments {
					clear(argument)
				}
				return err
			}
			remaining -= len(argument) + 32
			arguments = append(arguments, argument)
		}
		return inspectedCommand(arguments, writer, options.Inspect)
	}
	if options.Inspect != nil {
		if err := options.Inspect([][]byte{name}); err != nil {
			return err
		}
	}
	if err := WriteAll(writer, []byte("*"+strconv.FormatInt(count, 10)+"\r\n")); err != nil {
		return err
	}
	encodedName := EncodeCommand(name)
	firstHeader := bytes.IndexByte(encodedName, '\n') + 1
	if err := WriteAll(writer, encodedName[firstHeader:]); err != nil {
		return err
	}
	for index := int64(1); index < count; index++ {
		kind, err := reader.ReadByte()
		if err != nil {
			return err
		}
		if kind != '$' {
			return InvalidFrame
		}
		header, err := readLine(reader, options.MaximumHeaderBytes)
		if err != nil {
			return err
		}
		length, err := decimal(header)
		if err != nil {
			return err
		}
		if length < 0 {
			return InvalidFrame
		}
		if length > options.MaximumBulkBytes {
			return LimitExceeded
		}
		if err := WriteAll(writer, append(append([]byte{'$'}, header...), '\r', '\n')); err != nil {
			return err
		}
		if _, err := io.CopyN(writer, reader, length); err != nil {
			return err
		}
		var ending [2]byte
		if _, err := io.ReadFull(reader, ending[:]); err != nil {
			return err
		}
		if ending[0] != '\r' || ending[1] != '\n' {
			return InvalidFrame
		}
		if err := WriteAll(writer, ending[:]); err != nil {
			return err
		}
	}
	return nil
}
