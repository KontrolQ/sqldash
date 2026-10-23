package overviewhelper

import (
	"context"
	"time"

	"sqldash/config"
	"sqldash/database"
	"sqldash/engines"
	"sqldash/models"
	"sqldash/pgcluster/control"
	"sqldash/services/access"
	primarycredentials "sqldash/services/postgresconnections/primary"
)

func Current(ctx context.Context, record *models.Database) (*Credentials, error) {
	accountID := access.AccountID(ctx)
	fresh, current, authorityError := authorize(ctx, record, accountID)
	if authorityError != nil {
		return nil, authorityError
	}
	connection, findError := primarycredentials.Connection(fresh, current)
	if findError != nil {
		return nil, primarycredentials.Unavailable
	}
	if _, _, issuerError := authorize(fresh, current, connection.AccountID); issuerError != nil {
		return nil, primarycredentials.Unavailable
	}
	password, passwordError := primarycredentials.Password(current, connection)
	if passwordError != nil {
		return nil, primarycredentials.Unavailable
	}
	if verifyError := control.VerifyLoginPassword(fresh, current, connection, password); verifyError != nil {
		return nil, primarycredentials.Unavailable
	}
	fresh, current, authorityError = authorize(fresh, current, accountID)
	if authorityError != nil {
		return nil, authorityError
	}
	if _, _, issuerError := authorize(fresh, current, connection.AccountID); issuerError != nil {
		return nil, primarycredentials.Unavailable
	}
	verified, currentError := primarycredentials.Connection(fresh, current)
	if currentError != nil || verified.ID != connection.ID || verified.Generation != connection.Generation || verified.AccountID != connection.AccountID || verified.PasswordReference != connection.PasswordReference || !verified.Usable(time.Now()) {
		return nil, primarycredentials.Unavailable
	}
	return &Credentials{Identifier: verified.ID, Username: verified.Username, Password: password, URL: PasswordURL(config.PostgresHost(), config.Postgres.PublicPort, current.Name, verified.Username, password)}, nil
}

func authorize(ctx context.Context, record *models.Database, accountID uint) (context.Context, *models.Database, error) {
	actor, authorityError := primarycredentials.Authorize(ctx, database.DB.WithContext(ctx), record, accountID)
	if authorityError != nil {
		return ctx, nil, primarycredentials.Refused
	}
	var current models.Database
	findError := access.Scope(database.DB.WithContext(ctx).Model(&models.Database{}), actor).Where("id = ? AND name = ?", record.ID, record.Name).First(&current).Error
	if findError != nil || current.Kind() != models.EnginePostgres || !current.Available() {
		return ctx, nil, primarycredentials.Refused
	}
	return engines.WithDatabase(access.WithContext(ctx, actor), &current), &current, nil
}
