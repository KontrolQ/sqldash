package pgtransfer

import (
	"sqldash/models"
	"sqldash/pgcluster/control"
)

func (self *job) validatePublicNamespace(record *models.Database, credential *control.TransferCredential) error {
	if validateTarget(record) != nil {
		return InvalidTarget
	}
	connection, connectError := self.privateConnection(record, credential)
	if connectError != nil {
		return connectError
	}
	defer closeSourceConnection(connection)
	roles, roleError := control.DatabaseRoles(record)
	if roleError != nil {
		return InvalidTarget
	}
	var owner string
	if queryError := connection.QueryRow(self.context, `SELECT owner.rolname FROM pg_catalog.pg_namespace AS namespace
		JOIN pg_catalog.pg_roles AS owner ON owner.oid=namespace.nspowner WHERE namespace.nspname='public'`).Scan(&owner); queryError != nil || owner != roles.Owner {
		return InvalidTarget
	}
	var existing bool
	queryError := connection.QueryRow(self.context, `SELECT EXISTS(
		SELECT 1 FROM pg_catalog.pg_class AS relation JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid=relation.relnamespace
		WHERE namespace.nspname='public' AND relation.relkind IN ('r','p','v','m','S','f')
		AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_depend AS dependency WHERE dependency.classid='pg_catalog.pg_class'::regclass
		AND dependency.objid=relation.oid AND dependency.refclassid='pg_catalog.pg_extension'::regclass AND dependency.deptype='e')
		UNION ALL
		SELECT 1 FROM pg_catalog.pg_proc AS routine JOIN pg_catalog.pg_namespace AS namespace ON namespace.oid=routine.pronamespace
		WHERE namespace.nspname='public' AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_depend AS dependency
		WHERE dependency.classid='pg_catalog.pg_proc'::regclass AND dependency.objid=routine.oid
		AND dependency.refclassid='pg_catalog.pg_extension'::regclass AND dependency.deptype='e'))`).Scan(&existing)
	if queryError != nil {
		return self.safeError(InvalidTarget)
	}
	if existing {
		return InvalidTarget
	}
	return nil
}
