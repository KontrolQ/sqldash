package secretstore

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Store struct {
	directory string
	cipher    cipher.AEAD
	keyDigest [32]byte
	mutex     sync.Mutex
}

func Open(directory string) (*Store, error) {
	absoluteDirectory, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	if err = privateDirectory(absoluteDirectory); err != nil {
		return nil, err
	}
	store := &Store{directory: absoluteDirectory}
	err = store.locked(func() error {
		key, keyError := store.readOrCreateKey()
		if keyError != nil {
			return keyError
		}
		defer clear(key)
		store.keyDigest = sha256.Sum256(key)
		block, cipherError := aes.NewCipher(key)
		if cipherError != nil {
			return ErrInvalidKey
		}
		store.cipher, cipherError = cipher.NewGCM(block)
		if cipherError != nil {
			return cipherError
		}
		if cleanupError := store.removePendingFiles(); cleanupError != nil {
			return cleanupError
		}
		return store.verifySecrets()
	})
	if err != nil {
		return nil, err
	}
	return store, nil
}

func (self *Store) Set(namespace, name string, value []byte) error {
	if !validName(namespace) || !validName(name) || len(value) > maximumSecretBytes {
		return ErrInvalidName
	}
	return self.locked(func() error { return self.set(namespace, name, value) })
}

func (self *Store) Get(namespace, name string) ([]byte, error) {
	if !validName(namespace) || !validName(name) {
		return nil, ErrInvalidName
	}
	var value []byte
	err := self.locked(func() error {
		var readError error
		value, readError = self.get(namespace, name)
		return readError
	})
	return value, err
}

func (self *Store) Delete(namespace, name string) error {
	if !validName(namespace) || !validName(name) {
		return ErrInvalidName
	}
	return self.locked(func() error {
		path := self.path(namespace, name)
		if err := validatePrivateFile(path); errors.Is(err, os.ErrNotExist) {
			return nil
		} else if err != nil {
			return err
		}
		if err := os.Remove(path); err != nil {
			return err
		}
		return syncDirectory(filepath.Dir(path))
	})
}

func (self *Store) GetOrCreate(namespace, name string, create func() ([]byte, error)) ([]byte, error) {
	if !validName(namespace) || !validName(name) || create == nil {
		return nil, ErrInvalidName
	}
	var value []byte
	err := self.locked(func() error {
		var readError error
		value, readError = self.get(namespace, name)
		if !errors.Is(readError, ErrNotFound) {
			return readError
		}
		value, readError = create()
		if readError != nil {
			return readError
		}
		if len(value) > maximumSecretBytes {
			clear(value)
			return ErrInvalidName
		}
		if writeError := self.set(namespace, name, value); writeError != nil {
			clear(value)
			value = nil
			return writeError
		}
		return nil
	})
	return value, err
}

func (self *Store) locked(operation func() error) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if err := validateDirectory(self.directory); err != nil {
		return err
	}
	lockPath := filepath.Join(self.directory, lockFileName)
	if err := rejectSymlinkParents(lockPath); err != nil {
		return err
	}
	if information, err := os.Lstat(lockPath); err == nil && !information.Mode().IsRegular() {
		return ErrUnsafePath
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, FileMode)
	if err != nil {
		return err
	}
	defer lock.Close()
	if err = protectPath(lockPath, false); err != nil {
		return err
	}
	pathInformation, pathError := os.Lstat(lockPath)
	fileInformation, fileError := lock.Stat()
	if pathError != nil || fileError != nil || !os.SameFile(pathInformation, fileInformation) || !fileInformation.Mode().IsRegular() {
		return ErrUnsafePath
	}
	if err = lockFile(lock); err != nil {
		return err
	}
	defer unlockFile(lock)
	if self.cipher != nil {
		key, keyError := readPrivateFile(filepath.Join(self.directory, keyFileName))
		if keyError != nil {
			return keyError
		}
		matches := len(key) == KeySize && sha256.Sum256(key) == self.keyDigest
		clear(key)
		if !matches {
			return ErrInvalidKey
		}
	}
	return operation()
}

func (self *Store) readOrCreateKey() ([]byte, error) {
	keyPath := filepath.Join(self.directory, keyFileName)
	key, err := readPrivateFile(keyPath)
	if err == nil {
		if len(key) != KeySize {
			clear(key)
			return nil, ErrInvalidKey
		}
		return key, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	entries, err := os.ReadDir(self.directory)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.Name() != lockFileName {
			return nil, ErrMissingKey
		}
	}
	key = make([]byte, KeySize)
	if _, err = rand.Read(key); err != nil {
		return nil, err
	}
	if err = atomicWrite(keyPath, key); err != nil {
		clear(key)
		return nil, err
	}
	return key, nil
}

func (self *Store) path(namespace, name string) string {
	return filepath.Join(self.directory, namespace, name+secretExtension)
}

func (self *Store) get(namespace, name string) ([]byte, error) {
	if err := validateDirectory(filepath.Join(self.directory, namespace)); errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	sealed, err := readPrivateFile(self.path(namespace, name))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(sealed) < 4+self.cipher.NonceSize()+self.cipher.Overhead() || string(sealed[:4]) != "SQS1" {
		return nil, ErrCorruptSecret
	}
	value, err := self.cipher.Open(nil, sealed[4:4+self.cipher.NonceSize()], sealed[4+self.cipher.NonceSize():], []byte(namespace+"/"+name))
	if err != nil {
		return nil, ErrCorruptSecret
	}
	return value, nil
}

func (self *Store) set(namespace, name string, value []byte) error {
	directory := filepath.Join(self.directory, namespace)
	if err := privateDirectory(directory); err != nil {
		return err
	}
	path := self.path(namespace, name)
	if err := validatePrivateFile(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	nonce := make([]byte, self.cipher.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	sealed := append([]byte("SQS1"), nonce...)
	sealed = self.cipher.Seal(sealed, nonce, value, []byte(namespace+"/"+name))
	return atomicWrite(path, sealed)
}

func (self *Store) verifySecrets() error {
	return filepath.WalkDir(self.directory, func(path string, entry fs.DirEntry, walkError error) error {
		if walkError != nil {
			return walkError
		}
		if entry.IsDir() {
			return validateDirectory(path)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return ErrUnsafePath
		}
		if filepath.Dir(path) == self.directory && (entry.Name() == keyFileName || entry.Name() == lockFileName) {
			return validatePrivateFile(path)
		}
		if !strings.HasSuffix(entry.Name(), secretExtension) {
			return ErrUnsafePath
		}
		namespace := filepath.Base(filepath.Dir(path))
		name := strings.TrimSuffix(entry.Name(), secretExtension)
		if filepath.Dir(filepath.Dir(path)) != self.directory || !validName(namespace) || !validName(name) {
			return ErrUnsafePath
		}
		value, err := self.get(namespace, name)
		clear(value)
		return err
	})
}

func (self *Store) removePendingFiles() error {
	return filepath.WalkDir(self.directory, func(path string, entry fs.DirEntry, walkError error) error {
		if walkError != nil {
			return walkError
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return ErrUnsafePath
		}
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), ".pending-") {
			if err := validatePrivateFile(path); err != nil {
				return err
			}
			if err := os.Remove(path); err != nil {
				return err
			}
			return syncDirectory(filepath.Dir(path))
		}
		return nil
	})
}

func validName(value string) bool {
	if len(value) == 0 || len(value) > 128 || value[0] == '.' {
		return false
	}
	for _, character := range value {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || character == '-' || character == '_' || character == '.' {
			continue
		}
		return false
	}
	return true
}
