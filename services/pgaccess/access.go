package pgaccess

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"net"
	"sqldash/config"
	"sqldash/database"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/pgcluster/control"
	"sqldash/pggateway"
	"sqldash/services/access"
	"time"
)

var Refused = errors.New("the PostgreSQL connection is not authorized")

func Authorize(parent context.Context, username string, name string, address net.IP) (*pggateway.Identity, error) {
	if !config.Postgres.Enabled || address == nil || control.InternalRoleName(username) {
		return nil, Refused
	}
	var connection models.PostgresConnection
	if findError := database.DB.WithContext(parent).Where("username = ?", username).First(&connection).Error; findError != nil {
		return nil, Refused
	}
	if _, validateError := validate(parent, &connection, name, address); validateError != nil {
		return nil, validateError
	}
	return &pggateway.Identity{CredentialID: connection.ID, DatabaseID: connection.DatabaseID, AccountID: connection.AccountID, RoleName: connection.Username, DatabaseName: name, Generation: connection.Generation, SourceIP: address.String()}, nil
}

func Validate(parent context.Context, identity *pggateway.Identity) error {
	if identity == nil || !config.Postgres.Enabled || control.InternalRoleName(identity.RoleName) {
		return Refused
	}
	var connection models.PostgresConnection
	if findError := database.DB.WithContext(parent).First(&connection, identity.CredentialID).Error; findError != nil {
		return Refused
	}
	if connection.DatabaseID != identity.DatabaseID || connection.AccountID != identity.AccountID || connection.Username != identity.RoleName || connection.Generation != identity.Generation {
		return Refused
	}
	_, validateError := validate(parent, &connection, identity.DatabaseName, net.ParseIP(identity.SourceIP))
	return validateError
}

func validate(parent context.Context, connection *models.PostgresConnection, name string, address net.IP) (*models.Database, error) {
	if connection == nil || !connection.Usable(time.Now()) || address == nil || control.InternalRoleName(connection.Username) {
		return nil, Refused
	}
	var record models.Database
	store := database.DB.WithContext(parent)
	if findError := store.First(&record, connection.DatabaseID).Error; findError != nil || record.Kind() != models.EnginePostgres || !record.Available() || record.Name != name {
		return nil, Refused
	}
	issuer, loadError := access.LoadWith(store, connection.AccountID)
	if loadError != nil || !issuer.Can(permissions.PostgresCredentialsManage) || !issuer.Can(permissions.PostgresDataRead) {
		return nil, Refused
	}
	if connection.Scope != models.ScopeReadOnly && connection.Scope != models.ScopeReadWrite {
		return nil, Refused
	}
	if connection.Scope == models.ScopeReadWrite && !issuer.Can(permissions.PostgresDataWrite) {
		return nil, Refused
	}
	var visible int64
	if findError := access.Scope(store.Model(&models.Database{}), issuer).Where("id = ?", record.ID).Count(&visible).Error; findError != nil || visible != 1 {
		return nil, Refused
	}
	var rules []models.AccessRule
	if findError := store.Where("database_name = ?", record.Name).Find(&rules).Error; findError != nil {
		return nil, Refused
	}
	if len(rules) > 0 {
		allowed := false
		for _, rule := range rules {
			_, network, parseError := net.ParseCIDR(rule.Network)
			if parseError != nil {
				return nil, Refused
			}
			allowed = allowed || network.Contains(address)
		}
		if !allowed {
			return nil, Refused
		}
	}
	return &record, nil
}

func Terminate(parent context.Context, identity *pggateway.Identity, backendPID uint32) error {
	if identity == nil || backendPID == 0 || control.InternalRoleName(identity.RoleName) {
		return Refused
	}
	var record models.Database
	findError := database.DB.WithContext(parent).First(&record, identity.DatabaseID).Error
	if errors.Is(findError, gorm.ErrRecordNotFound) {
		return nil
	}
	if findError != nil {
		return findError
	}
	if record.Kind() != models.EnginePostgres || record.Name != identity.DatabaseName {
		return Refused
	}
	return control.Terminate(parent, &record, identity.RoleName, backendPID)
}
