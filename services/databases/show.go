package databases

import (
	"context"

	"sqldash/config"
	"sqldash/engines"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/pgcluster/control"
	ruleRepository "sqldash/repositories/accessrule"
	"sqldash/services/access"
	"sqldash/services/databases/overviewhelper"
	tokenservice "sqldash/services/tokens"
	"sqldash/utils/collections"

	"github.com/gofiber/fiber/v3"
)

func GetShowData(ctx context.Context, name, secret string) (*ShowContext, *fiber.Error) {
	fresh, record, err := Authorize(ctx, name, permissions.DatabaseView, false)
	if err != nil {
		return nil, err
	}
	actor := access.FromContext(fresh)
	address := record.Name + "." + config.Server.Domain
	shown := &ShowContext{Title: record.Label(), Name: record.Name, Engine: record.Kind(), EngineLabel: engineLabel(record.Kind()), State: stateLabel(record), Ready: record.Available(), Actions: access.Actions(actor, record), Address: address, URL: AddressScheme + address, Protected: record.Protected, Storage: "Unavailable", Tables: "—", RowsRead: "—", RowsWritten: "—", QueryCount: "—", Connections: "—", Transactions: "—", Scopes: collections.OptionsOf(ReadWriteValue, ReadWriteLabel, ReadOnlyValue, ReadOnlyLabel)}
	shown.CanCloneRemote = overviewhelper.CanCloneRemote(actor)
	shown.RemoteTLSModes = overviewhelper.RemoteTLSModes()
	if record.Kind() == models.EngineRedis {
		if err := redisContext(fresh, record, shown); err != nil {
			return nil, err
		}
	} else if record.Kind() == models.EnginePostgres {
		if record.Available() {
			if statistics, err := control.Statistics(fresh, record); err == nil {
				shown.Storage = readableSize(statistics.StorageBytes)
				shown.Connections = readableCount(statistics.Connections)
				shown.Transactions = readableCount(statistics.TransactionsCommitted)
				shown.Online = true
			}
			if actor.CanDatabase(record, permissions.DatabaseSettings) {
				if settings, err := control.GetSettings(fresh, record); err == nil {
					shown.SettingsAvailable = true
					shown.ConnectionLimit = settings.ConnectionLimit
					shown.StatementTimeoutMilliseconds = settings.StatementTimeoutMilliseconds
					shown.IdleTransactionTimeoutMilliseconds = settings.IdleTransactionTimeoutMilliseconds
				}
			}
		}
	} else if record.Available() {
		if statistics, err := engines.Statistics(fresh, name); err == nil {
			shown.Storage = readableSize(statistics.StorageBytesUsed)
			shown.RowsRead = readableCount(statistics.RowsRead)
			shown.RowsWritten = readableCount(statistics.RowsWritten)
			shown.QueryCount = readableCount(statistics.QueryCount)
			shown.Online = true
		}
		if settings, err := engines.Configuration(fresh, name); err == nil {
			shown.SettingsAvailable = true
			shown.BlockReads, shown.BlockWrites, shown.AllowAttach = settings.BlockReads, settings.BlockWrites, settings.AllowAttach
			if settings.BlockReason != nil {
				shown.BlockReason = *settings.BlockReason
			}
		}
		if actor.CanDatabase(record, permissions.TokenManage) {
			shown.Secret = secret
			tokenViews, err := tokenservice.ForDatabase(name)
			if err != nil {
				return nil, err
			}
			shown.Tokens = tokenViews
		}
	}
	if record.Kind() != models.EngineRedis && record.Available() && actor.CanDatabase(record, permissions.DataRead) {
		shown.Tables = tableCount(fresh, record)
	}
	if actor.CanDatabase(record, permissions.NetworkManage) {
		if rules, err := ruleRepository.ForDatabase(name); err == nil {
			for _, rule := range rules {
				shown.Rules = append(shown.Rules, RuleView{Identifier: rule.ID, Network: rule.Network, Note: rule.Note})
			}
		}
	}
	if record.Kind() == models.EnginePostgres {
		if credentialError := postgresContext(fresh, record, shown); credentialError != nil {
			return nil, credentialError
		}
	}
	return shown, nil
}
