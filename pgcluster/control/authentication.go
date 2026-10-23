package control

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
)

func authenticatedConnect(ctx context.Context, configuration *pgx.ConnConfig) (*pgx.Conn, error) {
	if configuration.RequireAuth != "scram-sha-256" || configuration.ValidateConnect != nil || configuration.AfterConnect != nil {
		return nil, AuthenticationPolicyMismatch
	}
	bounded, cancel := context.WithTimeout(ctx, ConnectionTimeout)
	defer cancel()
	connection, connectError := pgx.ConnectConfig(bounded, configuration)
	if connectError != nil {
		if strings.Contains(connectError.Error(), "require_auth check failed") {
			return nil, AuthenticationPolicyMismatch
		}
		return nil, SafeError(connectError)
	}
	return connection, nil
}
