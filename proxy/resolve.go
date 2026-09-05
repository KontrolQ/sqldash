package proxy

import (
	"net"
	"strings"

	"sqldash/config"
)

func databaseFor(request *incoming) string {
	if named := strings.TrimSpace(request.header.Get(NamespaceHeader)); named != "" {
		return named
	}

	host := hostWithoutPort(request.host)
	domain := strings.ToLower(config.Server.Domain)

	if host == domain {
		return ""
	}

	if !strings.HasSuffix(host, "."+domain) {
		return ""
	}

	label := strings.TrimSuffix(host, "."+domain)

	if strings.Contains(label, ".") {
		return ""
	}

	return label
}

func hostWithoutPort(host string) string {
	held := strings.ToLower(strings.TrimSpace(host))

	if trimmed, _, splitError := net.SplitHostPort(held); splitError == nil {
		return trimmed
	}

	return held
}
