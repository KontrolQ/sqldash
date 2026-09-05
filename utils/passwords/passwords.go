package passwords

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

func Hash(plain string) (string, error) {
	salt := make([]byte, SaltLength)

	if _, randomError := rand.Read(salt); randomError != nil {
		return "", errors.New(RandomFailed)
	}

	key := argon2.IDKey([]byte(plain), salt, TimeCost, MemoryCost, Threads, KeyLength)

	return fmt.Sprintf(
		Format,
		argon2.Version,
		MemoryCost,
		TimeCost,
		Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func Matches(plain string, stored string) bool {
	parts := strings.Split(stored, Separator)

	if len(parts) != FieldCount || parts[AlgorithmField] != Algorithm {
		return false
	}

	var version int

	if _, scanError := fmt.Sscanf(parts[VersionField], VersionFormat, &version); scanError != nil {
		return false
	}

	var (
		memoryCost uint32
		timeCost   uint32
		threads    uint8
	)

	read, scanError := fmt.Sscanf(parts[CostField], CostFormat, &memoryCost, &timeCost, &threads)
	if scanError != nil || read != CostFieldCount {
		return false
	}

	salt, saltError := base64.RawStdEncoding.DecodeString(parts[SaltField])
	if saltError != nil {
		return false
	}

	key, keyError := base64.RawStdEncoding.DecodeString(parts[KeyField])
	if keyError != nil || len(key) == 0 {
		return false
	}

	candidate := argon2.IDKey([]byte(plain), salt, timeCost, memoryCost, threads, uint32(len(key)))

	return subtle.ConstantTimeCompare(candidate, key) == 1
}

func Acceptable(plain string) error {
	if len(plain) < MinimumSize {
		return errors.New(TooShort)
	}

	return nil
}
