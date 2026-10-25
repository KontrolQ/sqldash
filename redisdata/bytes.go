package redisdata

import (
	"encoding/base64"
	"unicode/utf8"
)

func EncodeBytes(value []byte) Bytes {
	encoded := Bytes{Encoding: "base64", Data: base64.StdEncoding.EncodeToString(value)}
	if utf8.Valid(value) {
		text := string(value)
		encoded.Text = &text
	}
	return encoded
}

func DecodeBytes(value Bytes) ([]byte, error) {
	if value.Encoding != "base64" || len(value.Data) > base64.StdEncoding.EncodedLen(maximumBytes) {
		return nil, InvalidInput
	}
	decoded, err := base64.StdEncoding.Strict().DecodeString(value.Data)
	if err != nil || base64.StdEncoding.EncodeToString(decoded) != value.Data {
		return nil, InvalidInput
	}
	if value.Text != nil && (!utf8.Valid(decoded) || *value.Text != string(decoded)) {
		return nil, InvalidInput
	}
	return decoded, nil
}

func validKey(key []byte) error {
	if len(key) > maximumKeyBytes {
		return LimitExceeded
	}
	return nil
}
