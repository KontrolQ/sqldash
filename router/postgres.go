package router

import (
	databaseControllers "sqldash/controllers/databases"
	connectionControllers "sqldash/controllers/postgresconnections"
	managementControllers "sqldash/controllers/postgresmanagement"
	managementPages "sqldash/pages/postgresmanagement"
	"sqldash/utils/urls"
)

func init() {
	urls.SetNamespace("")
	urls.Path(urls.Post, "/databases/import/remote", databaseControllers.CloneRemote, "databases.import.remote")
	urls.Path(urls.Get, "/databases/:name/connections", managementPages.Connections, "databases.connections")
	urls.Path(urls.Post, "/databases/:name/connections/rotate", connectionControllers.RotatePrimary, "databases.connections.rotate.primary")
	urls.Path(urls.Post, "/databases/:name/connections/reveal", connectionControllers.Reveal, "databases.connections.reveal")
	urls.Path(urls.Post, "/databases/:name/connections", connectionControllers.Create, "databases.connections.create")
	urls.Path(urls.Post, "/databases/:name/connections/:identifier/rotate", connectionControllers.Rotate, "databases.connections.rotate")
	urls.Path(urls.Post, "/databases/:name/connections/:identifier/revoke", connectionControllers.Revoke, "databases.connections.revoke")
	urls.Path(urls.Post, "/databases/:name/settings/postgres", databaseControllers.SavePostgresSettings, "databases.settings.postgres")
	urls.Path(urls.Get, "/databases/:name/export/postgres/archive", databaseControllers.ExportPostgresArchive, "databases.export.postgres.archive")
	urls.Path(urls.Get, "/databases/:name/export/postgres/sql", databaseControllers.ExportPostgresSQL, "databases.export.postgres.sql")
	urls.Path(urls.Get, "/databases/:name/activity", managementPages.Activity, "databases.activity")
	urls.Path(urls.Post, "/databases/:name/activity/cancel", managementControllers.CancelActivity, "databases.activity.cancel")
	urls.Path(urls.Post, "/databases/:name/activity/terminate", managementControllers.TerminateActivity, "databases.activity.terminate")
	urls.Path(urls.Get, "/databases/:name/extensions", managementPages.Extensions, "databases.extensions")
	urls.Path(urls.Post, "/databases/:name/extensions", managementControllers.ManageExtension, "databases.extensions.manage")
	urls.Path(urls.Get, "/postgres/ca", publicPostgresCertificate, "postgres.certificate")
}
