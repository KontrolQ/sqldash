package tokens

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
)

type Minted struct {
	Secret     string
	Identifier string
	Digest     string
	Prefix     string
}

func Mint() (*Minted, error) {
	secret := make([]byte, SecretLength)
	if _, randomError := rand.Read(secret); randomError != nil {
		return nil, errors.New(RandomFailed)
	}

	identifier := make([]byte, IdentifierLength)
	if _, randomError := rand.Read(identifier); randomError != nil {
		return nil, errors.New(RandomFailed)
	}

	shown := base64.RawURLEncoding.EncodeToString(secret)

	return &Minted{
		Secret:     shown,
		Identifier: hex.EncodeToString(identifier),
		Digest:     Digest(shown),
		Prefix:     shown[:PrefixLength],
	}, nil
}

func Digest(secret string) string {
	sum := sha256.Sum256([]byte(secret))

	return hex.EncodeToString(sum[:])
}

func Same(offered string, stored string) bool {
	return subtle.ConstantTimeCompare([]byte(Digest(offered)), []byte(stored)) == 1
}
