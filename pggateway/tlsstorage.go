package pggateway

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func lockCertificateStorage(directory string) (func(), error) {
	lockPath := filepath.Join(directory, ".issuance.lock")
	lock, lockError := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if lockError != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidTLS, certificateStorageBusy)
	}
	if closeError := lock.Close(); closeError != nil {
		_ = os.Remove(lockPath)
		return nil, closeError
	}
	return func() { _ = os.Remove(lockPath) }, nil
}

func readCertificateFile(path string) ([]byte, error) {
	information, informationError := os.Lstat(path)
	if informationError != nil {
		return nil, informationError
	}
	if !information.Mode().IsRegular() || information.Size() > 1024*1024 {
		return nil, ErrInvalidTLS
	}
	file, openError := os.Open(path)
	if openError != nil {
		return nil, openError
	}
	defer file.Close()
	contents, readError := io.ReadAll(io.LimitReader(file, 1024*1024+1))
	if readError != nil || len(contents) > 1024*1024 {
		return nil, ErrInvalidTLS
	}
	return contents, nil
}

func certificatePairExists(certificatePath, keyPath string) (bool, error) {
	_, certificateError := os.Lstat(certificatePath)
	_, keyError := os.Lstat(keyPath)
	if os.IsNotExist(certificateError) && os.IsNotExist(keyError) {
		return false, nil
	}
	if certificateError != nil || keyError != nil {
		return false, fmt.Errorf("%w: %s", ErrInvalidTLS, incompleteCertificatePair)
	}
	return true, nil
}

func writeCertificatePair(certificatePath, keyPath string, publicCertificate, privateKey []byte) error {
	pendingPath := certificatePath + ".pending"
	if writeError := writeCertificateFile(pendingPath, publicCertificate, 0644); writeError != nil {
		return writeError
	}
	if writeError := writeCertificateFile(keyPath, privateKey, 0600); writeError != nil {
		return writeError
	}
	return os.Rename(pendingPath, certificatePath)
}

func writeCertificateFile(path string, contents []byte, mode os.FileMode) error {
	temporary, createError := os.CreateTemp(filepath.Dir(path), ".certificate-*")
	if createError != nil {
		return createError
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if permissionError := temporary.Chmod(mode); permissionError != nil {
		_ = temporary.Close()
		return permissionError
	}
	if writeError := writeAll(temporary, contents); writeError != nil {
		_ = temporary.Close()
		return writeError
	}
	if syncError := temporary.Sync(); syncError != nil {
		_ = temporary.Close()
		return syncError
	}
	if closeError := temporary.Close(); closeError != nil {
		return closeError
	}
	return os.Rename(temporaryPath, path)
}
