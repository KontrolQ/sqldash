package control

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/text/secure/precis"
)

func GeneratePassword() (string, error) {
	password := make([]byte, PasswordBytes)
	if _, randomError := rand.Read(password); randomError != nil {
		return "", randomError
	}
	defer clear(password)
	return base64.RawURLEncoding.EncodeToString(password), nil
}

func passwordVerifier(password string) (string, error) {
	if len(password) == 0 || strings.ContainsRune(password, 0) {
		return "", InvalidPassword
	}
	if normalized, normalizeError := precis.OpaqueString.String(password); normalizeError == nil {
		password = normalized
	}
	salt := make([]byte, 16)
	if _, randomError := rand.Read(salt); randomError != nil {
		return "", randomError
	}
	saltedPassword := pbkdf2.Key([]byte(password), salt, SCRAMIterations, sha256.Size, sha256.New)
	defer clear(saltedPassword)
	clientKey := hmacKey(saltedPassword, "Client Key")
	defer clear(clientKey)
	storedKey := sha256.Sum256(clientKey)
	serverKey := hmacKey(saltedPassword, "Server Key")
	defer clear(serverKey)
	return fmt.Sprintf("SCRAM-SHA-256$%d:%s$%s:%s", SCRAMIterations,
		base64.StdEncoding.EncodeToString(salt), base64.StdEncoding.EncodeToString(storedKey[:]), base64.StdEncoding.EncodeToString(serverKey)), nil
}

func hmacKey(key []byte, message string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(message))
	return mac.Sum(nil)
}

func pgLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
