package redishost

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"sqldash/config"
	"sqldash/models"
)

func validSettings(instance *models.RedisInstance) bool {
	if instance.MemoryLimit < 0 || instance.MemoryLimit > 0 && instance.MemoryLimit < 1024*1024 || instance.MemoryLimit > math.MaxInt64-instanceOverhead || memoryExceedsBudget(instance.MemoryLimit) {
		return false
	}
	switch instance.EvictionPolicy {
	case "noeviction", "allkeys-lru", "allkeys-lfu", "allkeys-random", "volatile-lru", "volatile-lfu", "volatile-random", "volatile-ttl":
	default:
		return false
	}
	if instance.Persistence != "aof-rdb" && instance.Persistence != "aof" && instance.Persistence != "rdb" {
		return false
	}
	return instance.Fsync == "everysec" || instance.Fsync == "always"
}

func writeConfiguration(instance *models.RedisInstance) error {
	if !validSettings(instance) {
		return InvalidSettings
	}
	_, groupID, err := instanceCredentials(instance.DatabaseID)
	if err != nil {
		return err
	}
	var contents strings.Builder
	contents.WriteString("bind 127.0.0.1\nport 0\nprotected-mode yes\ndaemonize no\nsupervised no\ndatabases 1\n")
	fmt.Fprintf(&contents, "unixsocket %s\nunixsocketperm 600\ndir %s\naclfile %s\n", strconv.Quote(socketPath(instance.DatabaseID)), strconv.Quote(dataDirectory(instance.DatabaseID)), strconv.Quote(aclPath(instance.DatabaseID)))
	fmt.Fprintf(&contents, "maxmemory %d\nmaxmemory-policy %s\n", instance.MemoryLimit, instance.EvictionPolicy)
	if config.Redis.MaximumConnections > 0 {
		fmt.Fprintf(&contents, "maxclients %d\n", config.Redis.MaximumConnections+privateClientReserve)
	}
	contents.WriteString("dbfilename dump.rdb\nappendfilename appendonly.aof\nappenddirname appendonlydir\naof-load-truncated no\nstop-writes-on-bgsave-error yes\nrdbchecksum yes\nlogfile \"\"\nslowlog-log-slower-than 10000\nslowlog-max-len 128\n")
	if instance.Persistence == "rdb" {
		contents.WriteString("appendonly no\n")
	} else {
		fmt.Fprintf(&contents, "appendonly yes\nappendfsync %s\n", instance.Fsync)
	}
	if instance.Persistence == "aof" {
		contents.WriteString("save \"\"\n")
	} else {
		contents.WriteString("save 900 1\nsave 300 10\nsave 60 10000\n")
	}
	for _, name := range moduleFiles {
		path := filepath.Join(config.Redis.ModulesDirectory, name)
		if err := rejectSymlinks(path); err != nil {
			return err
		}
		information, err := os.Stat(path)
		if err != nil || !information.Mode().IsRegular() {
			return Unavailable
		}
		fmt.Fprintf(&contents, "loadmodule %s\n", strconv.Quote(path))
	}
	return ownedFile(configurationPath(instance.DatabaseID), []byte(contents.String()), 0o640, groupID)
}
