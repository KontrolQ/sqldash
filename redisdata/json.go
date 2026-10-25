package redisdata

import (
	"bytes"
	"encoding/json"
	"unicode/utf8"
)

func FormatJSON(value []byte) (string, error) {
	if len(value) > maximumBytes {
		return "", LimitExceeded
	}
	if !utf8.Valid(value) || !json.Valid(value) {
		return "", InvalidInput
	}
	depth, expansion := 0, len(value)
	quoted, escaped := false, false
	for _, character := range value {
		if quoted {
			if escaped {
				escaped = false
			} else if character == '\\' {
				escaped = true
			} else if character == '"' {
				quoted = false
			}
			continue
		}
		switch character {
		case '"':
			quoted = true
		case '{', '[':
			depth++
			expansion += depth*2 + 1
		case '}', ']':
			expansion += depth*2 + 1
			depth--
		case ',':
			expansion += depth*2 + 1
		case ':':
			expansion++
		}
		if depth > 64 || expansion > maximumBytes*4 {
			return "", LimitExceeded
		}
	}
	var formatted bytes.Buffer
	if err := json.Indent(&formatted, value, "", "  "); err != nil {
		return "", InvalidInput
	}
	return formatted.String(), nil
}
