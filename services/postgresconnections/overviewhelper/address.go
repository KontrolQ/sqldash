package overviewhelper

import (
	"net"
	"net/url"
	"strconv"
	"strings"
)

func Address(host string, port int) string {
	if port != DefaultPort {
		return net.JoinHostPort(strings.TrimSuffix(strings.TrimPrefix(host, "["), "]"), strconv.Itoa(port))
	}
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		return "[" + host + "]"
	}
	return host
}

func ConnectionURL(host string, port int, databaseName, username string) string {
	return connectionURL(host, port, databaseName, url.User(username))
}

func PasswordURL(host string, port int, databaseName, username, password string) string {
	return connectionURL(host, port, databaseName, url.UserPassword(username, password))
}

func connectionURL(host string, port int, databaseName string, user *url.Userinfo) string {
	address := &url.URL{Scheme: "postgresql", Host: Address(host, port), Path: "/" + databaseName}
	if user.Username() != "" {
		address.User = user
	}
	address.RawQuery = url.Values{"sslmode": {"verify-full"}, "channel_binding": {"disable"}}.Encode()
	return address.String()
}
