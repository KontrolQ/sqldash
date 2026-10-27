package pictures

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"image"
	"io"
	"os"
	"regexp"
	"runtime"
	"strings"

	"sqldash/config"
)

var storedFilenamePattern = regexp.MustCompile(`^[a-f0-9]{64}\.(jpeg|png|gif|webp)$`)

func StoredName(reference string) (string, string, error) {
	filename, present := strings.CutPrefix(reference, ReferencePrefix)
	if !present || !storedFilenamePattern.MatchString(filename) {
		return "", "", ReferenceInvalid
	}
	_, extension, _ := strings.Cut(filename, ".")
	return filename, "image/" + extension, nil
}

func Store(encoded string, mediaType string) (string, error) {
	if len(encoded) > base64.StdEncoding.EncodedLen(MaximumStoredBytes) {
		return "", ImageTooComplex
	}
	body, decodingError := base64.StdEncoding.Strict().DecodeString(encoded)
	if decodingError != nil || len(body) == 0 {
		return "", ImageUnreadable
	}
	configuration, format, inspectionError := image.DecodeConfig(bytes.NewReader(body))
	if format == "apng" {
		format = "png"
	}
	if inspectionError != nil || mediaTypeOf(format) != mediaType || mediaType == "" || configuration.Width <= 0 || configuration.Height <= 0 || int64(configuration.Width)*int64(configuration.Height) > MaximumDecodedPixels {
		return "", ImageUnreadable
	}
	digest := sha256.Sum256(body)
	filename := hex.EncodeToString(digest[:]) + "." + format
	reference := ReferencePrefix + filename
	if directoryError := os.MkdirAll(config.PicturesPath(), DirectoryPermissions); directoryError != nil {
		return "", directoryError
	}
	root, openingError := os.OpenRoot(config.PicturesPath())
	if openingError != nil {
		return "", openingError
	}
	defer root.Close()
	if verificationError := verifyStored(root, filename, digest); verificationError == nil {
		return reference, syncStoredDirectory(root)
	} else if !errors.Is(verificationError, os.ErrNotExist) {
		return "", verificationError
	}

	var nonce [16]byte
	if _, randomnessError := rand.Read(nonce[:]); randomnessError != nil {
		return "", randomnessError
	}
	temporary := "pending-" + hex.EncodeToString(nonce[:])
	file, creationError := root.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, FilePermissions)
	if creationError != nil {
		return "", creationError
	}
	defer root.Remove(temporary)
	_, writingError := file.Write(body)
	if writingError == nil {
		writingError = file.Sync()
	}
	closingError := file.Close()
	if writingError != nil || closingError != nil {
		return "", errors.Join(writingError, closingError)
	}
	if linkingError := root.Link(temporary, filename); linkingError != nil && !errors.Is(linkingError, os.ErrExist) {
		return "", linkingError
	}
	if verificationError := verifyStored(root, filename, digest); verificationError != nil {
		return "", verificationError
	}
	return reference, syncStoredDirectory(root)
}

func syncStoredDirectory(root *os.Root) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	directory, openingError := root.Open(".")
	if openingError != nil {
		return openingError
	}
	syncError := directory.Sync()
	closingError := directory.Close()
	return errors.Join(syncError, closingError)
}

func verifyStored(root *os.Root, filename string, expected [sha256.Size]byte) error {
	file, openingError := root.Open(filename)
	if openingError != nil {
		return openingError
	}
	defer file.Close()
	information, inspectionError := file.Stat()
	if inspectionError != nil || !information.Mode().IsRegular() || information.Size() > MaximumStoredBytes {
		return StoredPictureDamaged
	}
	digest := sha256.New()
	if _, readingError := io.Copy(digest, file); readingError != nil {
		return readingError
	}
	if !bytes.Equal(digest.Sum(nil), expected[:]) {
		return StoredPictureDamaged
	}
	return nil
}

func Open(reference string) (*os.File, string, error) {
	filename, mediaType, validationError := StoredName(reference)
	if validationError != nil {
		return nil, "", validationError
	}
	root, openingError := os.OpenRoot(config.PicturesPath())
	if openingError != nil {
		return nil, "", openingError
	}
	defer root.Close()
	file, openingError := root.Open(filename)
	if openingError != nil {
		return nil, "", openingError
	}
	information, inspectionError := file.Stat()
	if inspectionError != nil || !information.Mode().IsRegular() || information.Size() > MaximumStoredBytes {
		file.Close()
		return nil, "", StoredPictureDamaged
	}
	return file, mediaType, nil
}
