package databases

import (
	"context"
	"fmt"
	"net/http"

	"sqldash/config"
	repository "sqldash/repositories/database"
	tokenservice "sqldash/services/tokens"
	"sqldash/sqld"
	"sqldash/utils/logger"
	"sqldash/utils/shortcuts"

	"github.com/gofiber/fiber/v3"
)

func GetShowData(requestContext context.Context, name string, secret string, problem string, done string) (*ShowContext, *fiber.Error) {
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
		Problem:   problem,
		Done:      done,
		Secret:    secret,
	}

	if held, statisticsError := sqld.Statistics(requestContext, name); statisticsError == nil {
		shown.Storage = readableSize(held.StorageBytesUsed)
		shown.RowsRead = held.RowsRead
		shown.RowsWritten = held.RowsWritten
		shown.QueryCount = held.QueryCount
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

	tokenViews, tokenError := tokenservice.ForDatabase(name)
	if tokenError != nil {
		return nil, tokenError
	}

	shown.Tokens = tokenViews

	return shown, nil
}

func readableSize(bytes int64) string {
	if bytes < KilobyteSize {
		return fmt.Sprintf(BytesFormat, bytes)
	}

	value := float64(bytes)
	units := []string{KilobyteUnit, MegabyteUnit, GigabyteUnit, TerabyteUnit}

	for _, unit := range units {
		value /= KilobyteSize

		if value < KilobyteSize {
			return fmt.Sprintf(SizeFormat, value, unit)
		}
	}

	return fmt.Sprintf(SizeFormat, value, TerabyteUnit)
}
