package databases

import (
	"crypto/rand"
	"math/big"
	"regexp"
	"strings"
)

var usernameNameSeparators = regexp.MustCompile(`[^a-z0-9]+`)

func creationNameCandidate(name string, username string, attempt int) (string, error) {
	if attempt == 0 {
		return name, nil
	}

	suffix := ""
	if attempt == 1 {
		suffix = strings.Trim(usernameNameSeparators.ReplaceAllString(strings.ToLower(strings.TrimSpace(username)), "-"), "-")
	}
	if suffix == "" {
		characters := make([]byte, NameSuffixLength)
		for index := range characters {
			position, randomError := rand.Int(rand.Reader, big.NewInt(int64(len(NameSuffixAlphabet))))
			if randomError != nil {
				return "", randomError
			}
			characters[index] = NameSuffixAlphabet[position.Int64()]
		}
		suffix = string(characters)
	}

	if len(suffix) > MaximumNameLength-MinimumNameLength-1 {
		suffix = strings.TrimRight(suffix[:MaximumNameLength-MinimumNameLength-1], "-")
	}
	prefixLength := min(len(name), MaximumNameLength-len(suffix)-1)
	prefix := strings.TrimRight(name[:prefixLength], "-")
	return prefix + "-" + suffix, nil
}
