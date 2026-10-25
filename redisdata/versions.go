package redisdata

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
)

type valueVersion struct {
	Type   string `json:"type"`
	Digest string `json:"digest"`
	Expiry string `json:"expiry"`
}

type readCursor struct {
	Type     string `json:"type"`
	Position string `json:"position"`
	Offset   int    `json:"offset"`
}

func requestScope(ctx context.Context) (Scope, error) {
	scope, present := ctx.Value(scopeContextKey{}).(Scope)
	if !present || scope.DatabaseID == "" || scope.Generation == "" || len(scope.DatabaseID) > 1024 || len(scope.Generation) > 1024 || len(scope.SigningKey) < 32 || len(scope.SigningKey) > 1024 {
		return Scope{}, InvalidScope
	}
	return scope, nil
}

func signature(scope Scope, purpose string, key, contents []byte) []byte {
	signer := hmac.New(sha256.New, scope.SigningKey)
	for _, part := range [][]byte{[]byte(scope.DatabaseID), []byte(scope.Generation), []byte(purpose), key, contents} {
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(len(part)))
		signer.Write(length[:])
		signer.Write(part)
	}
	return signer.Sum(nil)
}

func seal(scope Scope, purpose string, key []byte, payload any) string {
	contents, _ := json.Marshal(payload)
	return base64.RawURLEncoding.EncodeToString(contents) + "." + base64.RawURLEncoding.EncodeToString(signature(scope, purpose, key, contents))
}

func unseal(scope Scope, purpose string, key []byte, token string, payload any) error {
	if len(token) == 0 || len(token) > maximumTokenBytes {
		return InvalidVersion
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return InvalidVersion
	}
	contents, err := base64.RawURLEncoding.Strict().DecodeString(parts[0])
	if err != nil {
		return InvalidVersion
	}
	digest, err := base64.RawURLEncoding.Strict().DecodeString(parts[1])
	if err != nil || !hmac.Equal(digest, signature(scope, purpose, key, contents)) {
		return InvalidVersion
	}
	decoder := json.NewDecoder(strings.NewReader(string(contents)))
	decoder.DisallowUnknownFields()
	if decoder.Decode(payload) != nil {
		return InvalidVersion
	}
	return nil
}

func parseVersion(scope Scope, key []byte, token string) (valueVersion, error) {
	var version valueVersion
	if err := unseal(scope, "version", key, token, &version); err != nil {
		return version, err
	}
	digest, err := hex.DecodeString(version.Digest)
	if err != nil || len(digest) != 20 || version.Type == "none" || version.Type == "" {
		return version, InvalidVersion
	}
	expiry, err := strconv.ParseInt(version.Expiry, 10, 64)
	if err != nil || expiry < -1 || strconv.FormatInt(expiry, 10) != version.Expiry {
		return version, InvalidVersion
	}
	return version, nil
}

func boundedOptions(options ReadOptions) (ReadOptions, error) {
	if options.Limit == 0 {
		options.Limit = defaultLimit
	}
	if options.MaxBytes == 0 {
		options.MaxBytes = defaultBytes
	}
	if options.Limit < 1 || options.Limit > maximumLimit || options.MaxBytes < 1 || options.MaxBytes > maximumBytes || len(options.Cursor) > maximumTokenBytes {
		return options, LimitExceeded
	}
	return options, nil
}

func cursorFor(scope Scope, key []byte, token string) (readCursor, error) {
	cursor := readCursor{Position: "0"}
	if token == "" || token == "0" {
		return cursor, nil
	}
	if err := unseal(scope, "cursor", key, token, &cursor); err != nil {
		return cursor, err
	}
	if cursor.Offset < 0 || cursor.Offset > maximumLimit*16 || len(cursor.Position) > 64 {
		return cursor, InvalidInput
	}
	if cursor.Type == "stream" {
		if !validStreamID(cursor.Position, false) {
			return cursor, InvalidInput
		}
	} else if _, err := parseUnsigned(cursor.Position); err != nil {
		return cursor, err
	}
	return cursor, nil
}

func parseUnsigned(value string) (uint64, error) {
	number, err := strconv.ParseUint(value, 10, 64)
	if err != nil || strconv.FormatUint(number, 10) != value {
		return 0, InvalidInput
	}
	return number, nil
}
