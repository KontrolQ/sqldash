package pggateway

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"
)

type startupIdentity struct {
	username string
	database string
	protocol uint32
}

func parseStartup(packet startupPacket) (startupIdentity, error) {
	if packet.code != protocol30 && packet.code != protocol32 {
		return startupIdentity{}, ErrUnsupportedProtocol
	}
	parameters := packet.contents[8:]
	if len(parameters) == 0 || parameters[len(parameters)-1] != 0 {
		return startupIdentity{}, ErrInvalidStartup
	}
	values := make(map[string]string)
	for len(parameters) > 1 {
		nameEnd := bytes.IndexByte(parameters, 0)
		if nameEnd <= 0 {
			return startupIdentity{}, ErrInvalidStartup
		}
		name := string(parameters[:nameEnd])
		parameters = parameters[nameEnd+1:]
		valueEnd := bytes.IndexByte(parameters, 0)
		if valueEnd < 0 {
			return startupIdentity{}, ErrInvalidStartup
		}
		value := string(parameters[:valueEnd])
		parameters = parameters[valueEnd+1:]
		normalizedName := strings.ToLower(name)
		if _, duplicate := values[normalizedName]; duplicate || !knownStartupParameter(normalizedName) || !validStartupValue(value) {
			return startupIdentity{}, ErrInvalidStartup
		}
		if normalizedName == "user" || normalizedName == "database" {
			if name != normalizedName || !validIdentityName(value) {
				return startupIdentity{}, ErrInvalidStartup
			}
		}
		values[normalizedName] = value
	}
	if len(parameters) != 1 || parameters[0] != 0 || values["user"] == "" || values["database"] == "" {
		return startupIdentity{}, ErrInvalidStartup
	}
	return startupIdentity{username: values["user"], database: values["database"], protocol: packet.code}, nil
}

func knownStartupParameter(name string) bool {
	switch name {
	case "user", "database", "application_name", "client_encoding", "options", "datestyle", "timezone", "extra_float_digits", "search_path", "standard_conforming_strings", "default_transaction_read_only":
		return true
	default:
		return false
	}
}

func validStartupValue(value string) bool {
	if !utf8.ValidString(value) {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func validIdentityName(value string) bool {
	return len(value) > 0 && len(value) <= 63 && strings.TrimSpace(value) == value && validStartupValue(value)
}
