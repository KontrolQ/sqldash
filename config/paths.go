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

func PostgresDirectory() string {
	return filepath.Join(Data.Directory, "postgres")
}

func PostgresDataPath() string {
	return filepath.Join(PostgresDirectory(), "18", "data")
}

func PostgresConfigPath() string {
	return filepath.Join(PostgresDirectory(), "config")
}

func PostgresRuntimePath() string {
	return filepath.Join(PostgresDirectory(), "run")
}

func PostgresStagingPath() string {
	return filepath.Join(PostgresDirectory(), "staging")
}

func PostgresSecretsPath() string {
	return filepath.Join(PostgresDirectory(), "secrets")
}

func PostgresTLSPath() string {
	return filepath.Join(PostgresDirectory(), "tls")
}

func PostgresHost() string {
	if Postgres.PublicHost != "" {
		return Postgres.PublicHost
	}
	return Server.Domain
}

func RedisDirectory() string {
	return filepath.Join(Data.Directory, "redis")
}

func RedisInstancesPath() string {
	return filepath.Join(RedisDirectory(), "instances")
}

func RedisRuntimePath() string {
	return filepath.Join(RedisDirectory(), "run")
}

func RedisSecretsPath() string {
	return filepath.Join(RedisDirectory(), "secrets")
}

func RedisTLSPath() string {
	return filepath.Join(RedisDirectory(), "tls")
}

func RedisStagingPath() string {
	return filepath.Join(RedisDirectory(), "staging")
}

func RedisHost() string {
	if Redis.PublicHost != "" {
		return Redis.PublicHost
	}
	return Server.Domain
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
