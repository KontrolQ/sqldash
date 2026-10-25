package redishost

import (
	"path/filepath"
	"strconv"

	"sqldash/config"
)

func instanceName(databaseID uint) string {
	return strconv.FormatUint(uint64(databaseID), 10)
}

func stateDirectory() string {
	return filepath.Join(config.RedisDirectory(), "state")
}

func configurationDirectory(databaseID uint) string {
	return filepath.Join(config.RedisDirectory(), "config", instanceName(databaseID))
}

func dataDirectory(databaseID uint) string {
	return filepath.Join(config.RedisInstancesPath(), instanceName(databaseID))
}

func socketDirectory(databaseID uint) string {
	return filepath.Join(config.RedisDirectory(), "sockets", instanceName(databaseID))
}

func socketPath(databaseID uint) string {
	return filepath.Join(socketDirectory(databaseID), "redis.sock")
}

func manifestPath(databaseID uint) string {
	return filepath.Join(stateDirectory(), instanceName(databaseID)+".json")
}

func aclStatePath(databaseID uint) string {
	return filepath.Join(stateDirectory(), instanceName(databaseID)+".acl.json")
}

func aclPath(databaseID uint) string {
	return filepath.Join(configurationDirectory(databaseID), "users.acl")
}

func configurationPath(databaseID uint) string {
	return filepath.Join(configurationDirectory(databaseID), "redis.conf")
}

func controlSocketPath() string {
	return filepath.Join(config.RedisRuntimePath(), "control.sock")
}

func initializedPath() string {
	return filepath.Join(stateDirectory(), "initialized")
}

func lockPath(databaseID uint) string {
	return filepath.Join(stateDirectory(), instanceName(databaseID)+".lock")
}

func stateLockPath(databaseID uint) string {
	return filepath.Join(stateDirectory(), instanceName(databaseID)+".policy.lock")
}

func systemUsername(databaseID uint, profile Profile) string {
	return "sqd_" + instanceName(databaseID) + "_system_" + string(profile)
}

func secretNamespace(databaseID uint, identity string) string {
	return "instance-" + instanceName(databaseID) + "-" + identity
}
