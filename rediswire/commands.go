package rediswire

import (
	"bufio"
	"encoding/hex"
)

func ParseCommands(text string) ([][][]byte, error) {
	var commands [][][]byte
	var arguments [][]byte
	var current []byte
	var quote byte
	started := false
	closed := false
	flushArgument := func() error {
		if !started {
			return nil
		}
		arguments = append(arguments, append([]byte{}, current...))
		current = nil
		started = false
		closed = false
		return nil
	}
	flushCommand := func() error {
		if err := flushArgument(); err != nil {
			return err
		}
		if len(arguments) == 0 {
			return nil
		}
		commands = append(commands, arguments)
		arguments = nil
		return nil
	}
	for index := 0; index < len(text); index++ {
		character := text[index]
		if quote == '\'' {
			if character == '\\' && index+1 < len(text) && text[index+1] == '\'' {
				current = append(current, '\'')
				index++
				continue
			}
			if character == quote {
				quote = 0
				closed = true
			} else {
				current = append(current, character)
			}
			continue
		}
		if quote == 0 && (character == ' ' || character == '\t' || character == '\r' || character == '\n') {
			if character == '\n' {
				if err := flushCommand(); err != nil {
					return nil, err
				}
			} else if err := flushArgument(); err != nil {
				return nil, err
			}
			continue
		}
		if closed && quote == 0 {
			return nil, InvalidCommand
		}
		started = true
		if quote == 0 && (character == '\'' || character == '"') {
			quote = character
			continue
		}
		if quote == '"' && character == '"' {
			quote = 0
			closed = true
			continue
		}
		if character == '\\' {
			if index+1 >= len(text) {
				return nil, InvalidCommand
			}
			index++
			character = text[index]
			switch character {
			case 'n':
				character = '\n'
			case 'r':
				character = '\r'
			case 't':
				character = '\t'
			case 'b':
				character = '\b'
			case 'a':
				character = '\a'
			case 'x':
				if index+2 >= len(text) {
					return nil, InvalidCommand
				}
				var decoded [1]byte
				if _, err := hex.Decode(decoded[:], []byte(text[index+1:index+3])); err != nil {
					return nil, InvalidCommand
				}
				character = decoded[0]
				index += 2
			}
		}
		current = append(current, character)
	}
	if quote != 0 {
		return nil, InvalidCommand
	}
	if err := flushCommand(); err != nil {
		return nil, err
	}
	return commands, nil
}

func ReadCommand(reader *bufio.Reader, limits Limits) ([][]byte, error) {
	limits = normalizeLimits(limits)
	prefix, err := reader.Peek(1)
	if err != nil {
		return nil, err
	}
	if prefix[0] != '*' {
		line, err := readLine(reader, limits.MaximumLineBytes)
		if err != nil {
			return nil, err
		}
		if int64(len(line)+2) > limits.MaximumBytes {
			return nil, LimitExceeded
		}
		commands, err := ParseCommands(string(line))
		if err != nil || len(commands) != 1 {
			return nil, InvalidCommand
		}
		if len(commands[0]) > limits.MaximumElements {
			return nil, LimitExceeded
		}
		return commands[0], nil
	}
	value, err := ReadValue(reader, limits)
	if err != nil {
		return nil, err
	}
	if value.Null || len(value.Values) == 0 {
		return nil, InvalidCommand
	}
	arguments := make([][]byte, 0, len(value.Values))
	for _, argument := range value.Values {
		if argument.Null || argument.Kind != '$' && argument.Kind != '+' {
			return nil, InvalidCommand
		}
		arguments = append(arguments, argument.Bytes)
	}
	if len(arguments[0]) == 0 || len(arguments[0]) > 256 {
		return nil, InvalidCommand
	}
	return arguments, nil
}
