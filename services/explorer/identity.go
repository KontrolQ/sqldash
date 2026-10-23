package explorer

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"sqldash/config"
)

type rowValue struct {
	Name  string
	Kind  string
	Value string
	Null  bool
}

type rowIdentity struct {
	DatabaseID uint
	Schema     string
	Table      string
	Keys       []rowValue
	Revision   string
}

var identityKeyMutex sync.Mutex

func identityKey() ([]byte, error) {
	identityKeyMutex.Lock()
	defer identityKeyMutex.Unlock()
	path := filepath.Join(config.Data.Directory, "explorer.key")
	key, readError := os.ReadFile(path)
	if readError == nil {
		information, statError := os.Lstat(path)
		if statError != nil || len(key) != 32 || !information.Mode().IsRegular() || validateIdentityKey(path) != nil {
			return nil, errors.New("invalid row signing key")
		}
		return key, nil
	}
	if !errors.Is(readError, os.ErrNotExist) {
		return nil, readError
	}
	if directoryError := os.MkdirAll(config.Data.Directory, 0700); directoryError != nil {
		return nil, directoryError
	}
	key = make([]byte, 32)
	if _, randomError := rand.Read(key); randomError != nil {
		return nil, randomError
	}
	temporary, createError := os.CreateTemp(config.Data.Directory, ".explorer-key-*")
	if createError != nil {
		return nil, createError
	}
	defer os.Remove(temporary.Name())
	if permissionError := temporary.Chmod(0600); permissionError != nil {
		temporary.Close()
		return nil, permissionError
	}
	if protectionError := protectIdentityKey(temporary.Name()); protectionError != nil {
		temporary.Close()
		return nil, protectionError
	}
	if _, writeError := temporary.Write(key); writeError != nil {
		temporary.Close()
		return nil, writeError
	}
	if syncError := temporary.Sync(); syncError != nil {
		temporary.Close()
		return nil, syncError
	}
	if closeError := temporary.Close(); closeError != nil {
		return nil, closeError
	}
	if linkError := os.Link(temporary.Name(), path); linkError != nil {
		if !errors.Is(linkError, os.ErrExist) {
			return nil, linkError
		}
		key, readError = os.ReadFile(path)
		if readError != nil || len(key) != 32 || validateIdentityKey(path) != nil {
			return nil, errors.New("invalid row signing key")
		}
	}
	return key, nil
}

func signRow(databaseID uint, schema string, table string, columns []ColumnView, values []any, revision string) (string, error) {
	identity := rowIdentity{DatabaseID: databaseID, Schema: schema, Table: table, Revision: revision}
	for _, primary := range primaryColumns(columns) {
		for index, column := range columns {
			if primary.Name == column.Name {
				identity.Keys = append(identity.Keys, rowValue{Name: column.Name, Kind: column.Kind, Value: textOr(values[index], ""), Null: values[index] == nil})
			}
		}
	}
	encoded, encodeError := json.Marshal(identity)
	if encodeError != nil {
		return "", encodeError
	}
	key, keyError := identityKey()
	if keyError != nil {
		return "", keyError
	}
	signer := hmac.New(sha256.New, key)
	signer.Write(encoded)
	return base64.RawURLEncoding.EncodeToString(encoded) + "." + base64.RawURLEncoding.EncodeToString(signer.Sum(nil)), nil
}

func verifyRow(token string, databaseID uint, schema string, table string, columns []ColumnView) (*rowIdentity, error) {
	if len(token) > 128*1024 {
		return nil, errors.New("invalid row identity")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, errors.New("invalid row identity")
	}
	encoded, decodeError := base64.RawURLEncoding.DecodeString(parts[0])
	if decodeError != nil {
		return nil, decodeError
	}
	signature, decodeError := base64.RawURLEncoding.DecodeString(parts[1])
	if decodeError != nil {
		return nil, decodeError
	}
	key, keyError := identityKey()
	if keyError != nil {
		return nil, keyError
	}
	signer := hmac.New(sha256.New, key)
	signer.Write(encoded)
	if !hmac.Equal(signature, signer.Sum(nil)) {
		return nil, errors.New("invalid row signature")
	}
	var identity rowIdentity
	if decodeError := json.Unmarshal(encoded, &identity); decodeError != nil {
		return nil, decodeError
	}
	keys := primaryColumns(columns)
	if identity.DatabaseID != databaseID || identity.Schema != schema || identity.Table != table || identity.Revision == "" || len(keys) == 0 || len(keys) != len(identity.Keys) {
		return nil, errors.New("row identity does not match this table")
	}
	for index, key := range keys {
		if identity.Keys[index].Name != key.Name || identity.Keys[index].Kind != key.Kind || identity.Keys[index].Null {
			return nil, errors.New("row key changed")
		}
	}
	return &identity, nil
}

func identityPredicate(identity *rowIdentity, arguments *[]any) string {
	pieces := make([]string, 0, len(identity.Keys)+1)
	for _, key := range identity.Keys {
		*arguments = append(*arguments, key.Value)
		pieces = append(pieces, quoteIdentifier(key.Name)+" = "+parameterAt(len(*arguments))+"::text::"+key.Kind)
	}
	*arguments = append(*arguments, identity.Revision)
	pieces = append(pieces, "xmin = "+fmt.Sprintf("$%d::text::xid", len(*arguments)))
	return strings.Join(pieces, " AND ")
}
