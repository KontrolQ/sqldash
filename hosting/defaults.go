package hosting

import "time"

const (
	LogPrefix = "Hosting"

	RequestTimeout  = 90 * time.Second
	MaximumResponse = 1 << 20

	APIPrefix  = "/api/v2"
	LoginPath  = "/login"
	DomainPath = "/user/apps/appDefinitions/customdomain"
	SecurePath = "/user/apps/appDefinitions/enablecustomdomainssl"
	RemovePath = "/user/apps/appDefinitions/removecustomdomain"

	AppField      = "appName"
	DomainField   = "customDomain"
	PasswordField = "password"

	ContentTypeHeader = "Content-Type"
	ApplicationJSON   = "application/json"
	NamespaceHeader   = "x-namespace"
	CaptainNamespace  = "captain"
	AuthHeader        = "x-captain-auth"

	Accepted     = 100
	AlsoAccepted = 101
)
