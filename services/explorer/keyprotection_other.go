//go:build !windows

package explorer

import (
	"errors"
	"os"
)

func protectIdentityKey(path string) error { return os.Chmod(path, 0600) }

func validateIdentityKey(path string) error {
	information, statError := os.Lstat(path)
	if statError != nil {
		return statError
	}
	if !information.Mode().IsRegular() || information.Mode().Perm()&0077 != 0 {
		return errors.New("row signing key permissions are not protected")
	}
	return nil
}
