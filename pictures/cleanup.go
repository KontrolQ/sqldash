package pictures

import (
	"errors"
	"os"
	"regexp"

	"sqldash/config"
)

var pendingFilenamePattern = regexp.MustCompile(`^pending-[a-f0-9]{32}$`)

func Prune(references []string) error {
	root, openingError := os.OpenRoot(config.PicturesPath())
	if errors.Is(openingError, os.ErrNotExist) {
		return nil
	}
	if openingError != nil {
		return openingError
	}
	defer root.Close()
	retained := make(map[string]bool, len(references))
	for _, reference := range references {
		filename, _, validationError := StoredName(reference)
		if validationError == nil {
			retained[filename] = true
		}
	}
	directory, openingError := root.Open(".")
	if openingError != nil {
		return openingError
	}
	entries, readingError := directory.ReadDir(-1)
	directory.Close()
	if readingError != nil {
		return readingError
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() || retained[entry.Name()] || (!storedFilenamePattern.MatchString(entry.Name()) && !pendingFilenamePattern.MatchString(entry.Name())) {
			continue
		}
		if removalError := root.Remove(entry.Name()); removalError != nil {
			return removalError
		}
	}
	return nil
}
