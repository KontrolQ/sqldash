package sqld

import (
	"net/url"
	"strings"
)

func namespacePath(parts ...string) string {
	escaped := make([]string, 0, len(parts))

	for _, part := range parts {
		escaped = append(escaped, url.PathEscape(part))
	}

	return NamespacePrefix + "/" + strings.Join(escaped, "/")
}
