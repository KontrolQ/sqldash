package engines

import "errors"

var (
	UnknownEngine       = errors.New("the database engine is not supported")
	DatabaseUnavailable = errors.New("the database is not ready")
	Unsupported         = errors.New("this operation is not supported by the database engine")
	Conflict            = errors.New("the row changed; reload it before saving")
	PolicyUnconfirmed   = errors.New("database security reconciliation could not be confirmed")
)
