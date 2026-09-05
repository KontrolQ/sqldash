package proxy

import "time"

const (
	LogPrefix = "Proxy"

	NamespaceHeader       = "x-namespace"
	ContentEncodingHeader = "Content-Encoding"
	GzipEncoding          = "gzip"
	ForwardedForHeader    = "x-forwarded-for"
	ContentTypeHeader     = "content-type"
	UpgradeHeader         = "upgrade"
	WebsocketUpgrade      = "websocket"

	PipelinePathV2 = "/v2/pipeline"
	PipelinePathV3 = "/v3/pipeline"

	ReadHeaderTimeout = 10 * time.Second
	IdleTimeout       = 120 * time.Second
	ShutdownGrace     = 10 * time.Second

	MaximumMeasuredBody = 8 << 20

	RegistryFreshness = 5 * time.Second

	AuthorizationHeader = "authorization"
	BearerPrefix        = "bearer "
	UseNoteInterval     = time.Minute

	SocketScheme = "ws://"
	MaximumFrame = 32 << 20
)

const (
	StandardTLSPort = 443
	HTTPSScheme     = "https://"
)
