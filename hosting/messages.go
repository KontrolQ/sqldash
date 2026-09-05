package hosting

const (
	SignInFailedLog  = "Could not sign in to the hosting panel: %v"
	DomainFailedLog  = "Could not add %s: %v"
	SecureFailedLog  = "Could not get a certificate for %s: %v"
	RemoveFailedLog  = "Could not remove %s: %v"
	DomainAddedLog   = "%s now reaches this server"
	SecuredLog       = "%s is served over TLS"
	DomainRemovedLog = "%s no longer reaches this server"
)
