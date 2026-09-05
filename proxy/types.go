package proxy

import "net/http"

type incoming struct {
	host   string
	header http.Header
}

func incomingFrom(request *http.Request) *incoming {
	return &incoming{host: request.Host, header: request.Header}
}
