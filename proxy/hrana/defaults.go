package hrana

const (
	ExecuteRequest = "execute"
	BatchRequest   = "batch"
	OkResult       = "ok"

	RequestFrame       = "request"
	ResponseOkFrame    = "response_ok"
	ResponseErrorFrame = "response_error"
)

var Subprotocols = []string{"hrana3", "hrana2", "hrana1"}
