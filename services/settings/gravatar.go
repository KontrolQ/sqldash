package settings

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"

	repository "sqldash/repositories/account"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func UseGravatar(context context.Context, accountID uint) *fiber.Error {
	account, findError := repository.FindByID(accountID)
	if findError != nil || account == nil {
		logger.Errorf(LogPrefix, LookupFailedLog, findError)
		return shortcuts.ServiceError(http.StatusUnauthorized, AccountUnavailable)
	}

	hash := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(account.Email))))
	address := fmt.Sprintf(GravatarAddress, hash)
	request, requestError := http.NewRequestWithContext(context, http.MethodGet, address, nil)
	if requestError != nil {
		return shortcuts.ServiceError(http.StatusBadGateway, GravatarUnavailable)
	}
	client := &http.Client{Timeout: GravatarTimeout, CheckRedirect: func(request *http.Request, previous []*http.Request) error {
		if len(previous) >= 3 || request.URL.Scheme != "https" || (request.URL.Hostname() != "gravatar.com" && !strings.HasSuffix(request.URL.Hostname(), ".gravatar.com")) {
			return http.ErrUseLastResponse
		}
		return nil
	}}
	response, fetchError := client.Do(request)
	if fetchError != nil {
		logger.Errorf(LogPrefix, GravatarFailedLog, fetchError)
		return shortcuts.ServiceError(http.StatusBadGateway, GravatarUnavailable)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return shortcuts.ServiceError(http.StatusNotFound, GravatarMissing)
	}
	if response.StatusCode != http.StatusOK {
		return shortcuts.ServiceError(http.StatusBadGateway, GravatarUnavailable)
	}
	return SavePicture(accountID, response.Body)
}
