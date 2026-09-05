package telemetry

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"
)

func Normalise(statement string) string {
	var builder strings.Builder

	runes := []rune(statement)
	spaced := false

	for index := 0; index < len(runes); index++ {
		letter := runes[index]

		switch {
		case letter == '\'':
			index = skipQuoted(runes, index, '\'')
			builder.WriteByte('?')
			spaced = false
		case letter == '"':
			index = skipQuoted(runes, index, '"')
			builder.WriteByte('?')
			spaced = false
		case unicode.IsDigit(letter) && !spaced && startsNumber(runes, index):
			index = skipNumber(runes, index)
			builder.WriteByte('?')
		case unicode.IsSpace(letter):
			if !spaced && builder.Len() > 0 {
				builder.WriteByte(' ')
				spaced = true
			}
		default:
			builder.WriteRune(unicode.ToLower(letter))
			spaced = false
		}
	}

	return strings.TrimSpace(builder.String())
}

func Digest(normalised string) string {
	sum := sha256.Sum256([]byte(normalised))

	return hex.EncodeToString(sum[:])[:DigestLength]
}

func startsNumber(runes []rune, index int) bool {
	if index == 0 {
		return true
	}

	previous := runes[index-1]

	return !unicode.IsLetter(previous) && !unicode.IsDigit(previous) && previous != '_'
}

func skipNumber(runes []rune, index int) int {
	for index+1 < len(runes) && (unicode.IsDigit(runes[index+1]) || runes[index+1] == '.') {
		index++
	}

	return index
}

func skipQuoted(runes []rune, index int, quote rune) int {
	for index+1 < len(runes) {
		index++

		if runes[index] == quote {
			if index+1 < len(runes) && runes[index+1] == quote {
				index++
				continue
			}

			return index
		}
	}

	return index
}
