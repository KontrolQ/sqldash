package overviewhelper

import (
	"sqldash/config"
	"sqldash/permissions"
	"sqldash/pgtransfer"
	"sqldash/services/access"
	"sqldash/utils/collections"
)

func CanCloneRemote(actor *access.AccountAccess) bool {
	return config.Postgres.Enabled && actor.Can(permissions.PostgresDatabaseCreate) && actor.Can(permissions.PostgresDatabaseImport) && actor.Can(permissions.PostgresCredentialsManage) && actor.Can(permissions.PostgresDataRead) && actor.Can(permissions.PostgresDataWrite) && actor.Can(permissions.PostgresObjectsManage)
}

func RemoteTLSModes() []collections.Option {
	return collections.OptionsOf(pgtransfer.SSLVerifyFull, "Verify hostname and certificate", pgtransfer.SSLVerifyCA, "Verify certificate", pgtransfer.SSLRequire, "Encryption only", pgtransfer.SSLDisable, "No SSL (unencrypted)")
}
