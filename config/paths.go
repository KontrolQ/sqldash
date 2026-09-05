package config

import (
	"net"
	"path/filepath"
	"strconv"
)

func DatabasesPath() string {
	return filepath.Join(Data.Directory, DatabasesDirectory)
}

func ImportsPath() string {
	return filepath.Join(Data.Directory, ImportsDirectory)
}

func CertificatesPath() string {
	return filepath.Join(Data.Directory, CertificatesDirectory)
}

func StatePath() string {
	return filepath.Join(Data.Directory, StateFileName)
}

func SqldURL() string {
	return "http://" + Sqld.Address
}

func SqldAdminURL() string {
	return "http://" + Sqld.AdminAddress
}

func HTTPAddress() string {
	return net.JoinHostPort("", strconv.Itoa(Server.HTTPPort))
}

func HTTPSAddress() string {
	return net.JoinHostPort("", strconv.Itoa(Server.HTTPSPort))
}
