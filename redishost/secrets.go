package redishost

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"

	"sqldash/config"
	"sqldash/database"
	"sqldash/models"
	"sqldash/secretstore"
)

func randomSecret() ([]byte, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return nil, Unavailable
	}
	encoded := make([]byte, hex.EncodedLen(len(value)))
	hex.Encode(encoded, value)
	clear(value)
	return encoded, nil
}

func existingStore() (*secretstore.Store, error) {
	if _, err := readOwnedFile(initializedPath()); err != nil {
		return nil, Unavailable
	}
	if _, err := os.Lstat(filepath.Join(config.RedisSecretsPath(), "key")); err != nil {
		return nil, Unavailable
	}
	store, err := secretstore.Open(config.RedisSecretsPath())
	if err != nil {
		return nil, Unavailable
	}
	return store, nil
}

func prepareRoot() error {
	if !config.Redis.Enabled || !managedSupported() {
		return Unavailable
	}
	if err := prepareDirectories(); err != nil {
		return err
	}
	if _, err := os.Lstat(initializedPath()); err == nil {
		_, err := existingStore()
		return err
	} else if !errors.Is(err, os.ErrNotExist) {
		return UnsafePath
	}
	var existing int64
	if err := database.DB.Model(&models.RedisInstance{}).Count(&existing).Error; err != nil {
		return Unavailable
	}
	if existing > 0 {
		if _, err := os.Lstat(filepath.Join(config.RedisSecretsPath(), "key")); err != nil {
			return Unavailable
		}
	}
	store, err := secretstore.Open(config.RedisSecretsPath())
	if err != nil {
		return Unavailable
	}
	token, err := store.GetOrCreate("controller", "transport", randomSecret)
	clear(token)
	if err != nil {
		return Unavailable
	}
	return ownedFile(initializedPath(), []byte("1\n"), 0o600, 0)
}

func SecretStore() (*secretstore.Store, error) {
	return existingStore()
}

func controlToken() ([]byte, error) {
	store, err := existingStore()
	if err != nil {
		return nil, err
	}
	token, err := store.Get("controller", "transport")
	if err != nil || len(token) != 64 {
		clear(token)
		return nil, Unavailable
	}
	return token, nil
}

func systemPassword(instance *models.RedisInstance, profile Profile, create bool) ([]byte, error) {
	store, err := existingStore()
	if err != nil {
		return nil, err
	}
	namespace := secretNamespace(instance.DatabaseID, instance.Identity)
	var value []byte
	if create {
		value, err = store.GetOrCreate(namespace, string(profile), randomSecret)
	} else {
		value, err = store.Get(namespace, string(profile))
	}
	if err != nil || len(value) != 64 {
		clear(value)
		return nil, Unavailable
	}
	return value, nil
}
