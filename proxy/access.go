package proxy

import (
	"net"
	"net/http"
	"strings"
	"time"

	"sqldash/models"
	rulerepository "sqldash/repositories/accessrule"
	tokenrepository "sqldash/repositories/token"
	"sqldash/utils/logger"
	"sqldash/utils/tokens"
)

func permitted(request *http.Request, database string) (int, string) {
	if allowed, ruleError := addressAllowed(request, database); ruleError != nil {
		return http.StatusInternalServerError, AccessUnreadable
	} else if !allowed {
		return http.StatusForbidden, AddressRefused
	}

	offered := bearerFrom(request)
	if offered == "" {
		return http.StatusUnauthorized, TokenMissing
	}

	held, findError := tokenrepository.FindByDigest(tokens.Digest(offered))
	if findError != nil {
		logger.Errorf(LogPrefix, TokenLookupFailedLog, findError)
		return http.StatusInternalServerError, AccessUnreadable
	}

	if held == nil || held.DatabaseName != database || !held.Usable(time.Now()) {
		return http.StatusUnauthorized, TokenRefused
	}

	noteUse(held)

	return 0, ""
}

func bearerFrom(request *http.Request) string {
	held := request.Header.Get(AuthorizationHeader)

	if len(held) > len(BearerPrefix) && strings.EqualFold(held[:len(BearerPrefix)], BearerPrefix) {
		return strings.TrimSpace(held[len(BearerPrefix):])
	}

	return ""
}

func addressAllowed(request *http.Request, database string) (bool, error) {
	rules, findError := rulerepository.ForDatabase(database)
	if findError != nil {
		return false, findError
	}

	if len(rules) == 0 {
		return true, nil
	}

	address := callerAddress(request)
	if address == nil {
		return false, nil
	}

	for _, rule := range rules {
		_, network, parseError := net.ParseCIDR(strings.TrimSpace(rule.Network))
		if parseError != nil {
			continue
		}

		if network.Contains(address) {
			return true, nil
		}
	}

	return false, nil
}

func callerAddress(request *http.Request) net.IP {
	host, _, splitError := net.SplitHostPort(request.RemoteAddr)
	if splitError != nil {
		host = request.RemoteAddr
	}

	return net.ParseIP(strings.TrimSpace(host))
}

func noteUse(held *models.Token) {
	now := time.Now()

	if held.LastUsedAt != nil && now.Sub(*held.LastUsedAt) < UseNoteInterval {
		return
	}

	held.LastUsedAt = &now

	if saveError := tokenrepository.Save(held); saveError != nil {
		logger.Warnf(LogPrefix, TokenNoteFailedLog, saveError)
	}
}
