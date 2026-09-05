package sqld

import (
	"net/url"
	"path/filepath"
	"strings"
)

func namespacePath(parts ...string) string {
	escaped := make([]string, 0, len(parts))

	for _, part := range parts {
		escaped = append(escaped, url.PathEscape(part))
	}

	return NamespacePrefix + "/" + strings.Join(escaped, "/")
}

func dumpAddress(path string) string {
	held := filepath.ToSlash(path)

	if !strings.HasPrefix(held, "/") {
		held = "/" + held
	}

	return DumpScheme + held
}
