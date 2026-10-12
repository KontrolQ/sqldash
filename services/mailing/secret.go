package mailing

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"

	"sqldash/config"
)

var keyMutex sync.Mutex

func encryptionKey() ([]byte, error) {
	keyMutex.Lock()
	defer keyMutex.Unlock()
	path := filepath.Join(config.Data.Directory, KeyFilename)
	key, readError := os.ReadFile(path)
	if readError == nil {
		if len(key) != 32 {
			return nil, SecretUnavailable
		}
		return key, nil
	}
	if !errors.Is(readError, os.ErrNotExist) {
		return nil, readError
	}
	key = make([]byte, 32)
	if _, readError = rand.Read(key); readError != nil {
		return nil, readError
	}
	file, createError := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if errors.Is(createError, os.ErrExist) {
		return nil, SecretUnavailable
	}
	if createError != nil {
		return nil, createError
	}
	_, writeError := file.Write(key)
	closeError := file.Close()
	if writeError != nil {
		os.Remove(path)
		return nil, writeError
	}
	return key, closeError
}

func cryptography() (cipher.AEAD, error) {
	key, keyError := encryptionKey()
	if keyError != nil {
		return nil, keyError
	}
	block, blockError := aes.NewCipher(key)
	if blockError != nil {
		return nil, blockError
	}
	return cipher.NewGCM(block)
}

func encrypt(secret string) (string, error) {
	if secret == "" {
		return "", nil
	}
	codec, codecError := cryptography()
	if codecError != nil {
		return "", codecError
	}
	nonce := make([]byte, codec.NonceSize())
	if _, readError := io.ReadFull(rand.Reader, nonce); readError != nil {
		return "", readError
	}
	return base64.StdEncoding.EncodeToString(codec.Seal(nonce, nonce, []byte(secret), nil)), nil
}

func decrypt(secret string) (string, error) {
	if secret == "" {
		return "", nil
	}
	codec, codecError := cryptography()
	if codecError != nil {
		return "", codecError
	}
	data, decodeError := base64.StdEncoding.DecodeString(secret)
	if decodeError != nil || len(data) < codec.NonceSize() {
		return "", SecretUnavailable
	}
	plain, openError := codec.Open(nil, data[:codec.NonceSize()], data[codec.NonceSize():], nil)
	return string(plain), openError
}
