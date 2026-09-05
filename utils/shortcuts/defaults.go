package shortcuts

const (
	LogPrefix = "Shortcuts"

	HeaderHtmxRequest  = "HX-Request"
	HeaderHtmxBoosted  = "HX-Boosted"
	HeaderHtmxRedirect = "HX-Redirect"

	PartialTemplateFormat = "%s/htmx/%s.htmx"
	TemplateError         = "errors/error"
	SSEFormat             = "event: %s\ndata: %s\n\n"
)
