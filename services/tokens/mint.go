package tokens

import (
	"net/http"
	"strings"

	"sqldash/models"
	repository "sqldash/repositories/token"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"
	"sqldash/utils/tokens"

	"github.com/gofiber/fiber/v3"
)

func Mint(databaseName string, label string, scope string) (string, *fiber.Error) {
	wanted, scopeError := scopeFrom(scope)
	if scopeError != nil {
		return "", scopeError
	}

	label = strings.TrimSpace(label)
	if label == "" {
		label = DefaultLabel
	}

	if len(label) > MaximumLabel {
		label = label[:MaximumLabel]
	}

	minted, mintError := tokens.Mint()
	if mintError != nil {
		logger.Errorf(LogPrefix, MintFailedLog, databaseName, mintError)
		return "", shortcuts.ServiceError(http.StatusInternalServerError, MintRefused)
	}

	record := &models.Token{
		DatabaseName: databaseName,
		Label:        label,
		Identifier:   minted.Identifier,
		Digest:       minted.Digest,
		Prefix:       minted.Prefix,
		Scope:        wanted,
	}

	if createError := repository.Create(record); createError != nil {
		logger.Errorf(LogPrefix, MintFailedLog, databaseName, createError)
		return "", shortcuts.ServiceError(http.StatusInternalServerError, MintRefused)
	}

	return minted.Secret, nil
}
