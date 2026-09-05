package databases

import (
	"context"
	"net/http"

	"sqldash/config"
	ruleRepository "sqldash/repositories/accessrule"
	repository "sqldash/repositories/database"
	explorer "sqldash/services/explorer"
	tokenservice "sqldash/services/tokens"
	"sqldash/sqld"
	"sqldash/utils/collections"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func GetShowData(requestContext context.Context, name string, secret string) (*ShowContext, *fiber.Error) {
	held, findError := repository.FindByName(name)
	if findError != nil {
		logger.Errorf(LogPrefix, ListFailedLog, findError)
		return nil, shortcuts.ServiceError(http.StatusInternalServerError, ListUnavailable)
	}

	if held == nil {
		return nil, shortcuts.ServiceError(http.StatusNotFound, DatabaseMissing)
	}

	address := held.Name + "." + config.Server.Domain

	shown := &ShowContext{
		Title:     held.Name,
		Name:      held.Name,
		Address:   address,
		URL:       AddressScheme + address,
		Protected: held.Protected,
		Secret:    secret,
		Scopes:    collections.OptionsOf(ReadWriteValue, ReadWriteLabel, ReadOnlyValue, ReadOnlyLabel),
	}

	if held, statisticsError := sqld.Statistics(requestContext, name); statisticsError == nil {
		shown.Storage = readableSize(held.StorageBytesUsed)
		shown.RowsRead = readableCount(held.RowsRead)
		shown.RowsWritten = readableCount(held.RowsWritten)
		shown.QueryCount = readableCount(held.QueryCount)
	} else {
		logger.Warnf(LogPrefix, StatisticsFailedLog, name, statisticsError)
		shown.Storage = UnknownSize
	}

	if settings, settingsError := sqld.Configuration(requestContext, name); settingsError == nil {
		shown.BlockReads = settings.BlockReads
		shown.BlockWrites = settings.BlockWrites
		shown.AllowAttach = settings.AllowAttach

		if settings.BlockReason != nil {
			shown.BlockReason = *settings.BlockReason
		}
	} else {
		logger.Warnf(LogPrefix, ConfigurationFailedLog, name, settingsError)
	}

	if tables, tablesError := explorer.Tables(requestContext, name); tablesError == nil {
		shown.Tables = len(tables)
	}

	tokenViews, tokenError := tokenservice.ForDatabase(name)
	if tokenError != nil {
		return nil, tokenError
	}

	shown.Tokens = tokenViews

	if rules, rulesError := ruleRepository.ForDatabase(name); rulesError == nil {
		for _, rule := range rules {
			shown.Rules = append(shown.Rules, RuleView{
				Identifier: rule.ID,
				Network:    rule.Network,
				Note:       rule.Note,
			})
		}
	}

	return shown, nil
}
