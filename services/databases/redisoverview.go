package databases

import (
	"context"
	"net"
	"net/url"
	"strconv"
	"strings"

	"sqldash/config"
	"sqldash/database"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/redishost"
	"sqldash/services/access"
	"sqldash/services/redisconnections"
	"sqldash/utils/bytesize"
	"sqldash/utils/collections"

	"github.com/gofiber/fiber/v3"
)

func redisContext(ctx context.Context, record *models.Database, shown *ShowContext) *fiber.Error {
	shown.Host, shown.Port = config.RedisHost(), config.Redis.PublicPort
	shown.MemoryUsed = "—"
	shown.PlaintextAllowed = config.Redis.AllowPlaintext
	shown.RedisMemoryBudget = bytesize.FormatBinary(config.Redis.MemoryBudget)
	shown.RedisMemoryOverhead = bytesize.FormatBinary(redishost.InstanceMemoryOverhead)
	shown.ManagedRedisCA = config.Redis.CertificatePath == "" && !config.Certificates.Automatic
	shown.EvictionPolicies = collections.OptionsOf("noeviction", "No eviction", "allkeys-lru", "All keys · least recently used", "allkeys-lfu", "All keys · least frequently used", "allkeys-random", "All keys · random", "volatile-lru", "Expiring keys · least recently used", "volatile-lfu", "Expiring keys · least frequently used", "volatile-random", "Expiring keys · random", "volatile-ttl", "Expiring keys · shortest TTL")
	shown.PersistenceOptions = collections.OptionsOf("aof-rdb", "AOF and RDB snapshots", "aof", "AOF", "rdb", "RDB snapshots")
	shown.FsyncOptions = collections.OptionsOf("everysec", "Every second", "always", "Always")
	var primary models.RedisConnection
	if database.DB.WithContext(ctx).Select("username").Where("database_id = ? AND instance_identity = ? AND primary_connection = ?", record.ID, record.ClusterID, true).First(&primary).Error == nil {
		shown.Username = primary.Username
	}
	address := url.URL{Scheme: "rediss", Host: net.JoinHostPort(shown.Host, strconv.Itoa(shown.Port)), Path: "/0"}
	if shown.Username != "" {
		address.User = url.User(shown.Username)
	}
	shown.Address = address.String()
	shown.URL = shown.Address
	actor := access.FromContext(ctx)
	if instance, err := redishost.Instance(ctx, record); err == nil {
		shown.MemoryLimit = bytesize.FormatBinary(instance.MemoryLimit)
		shown.MemoryAmount, shown.MemoryUnit, _ = strings.Cut(shown.MemoryLimit, " ")
		shown.MemoryUnits = collections.OptionsOf("KiB", "KiB", "MiB", "MiB", "GiB", "GiB", "TiB", "TiB")
		shown.MemoryLimitBytes = strconv.FormatInt(instance.MemoryLimit, 10)
		shown.EvictionPolicy, shown.Persistence, shown.Fsync = instance.EvictionPolicy, instance.Persistence, instance.Fsync
		shown.SettingsAvailable = config.Redis.Enabled && instance.State == models.DatabaseReady
	}
	if record.Available() && config.Redis.Enabled {
		if statistics, err := redishost.StatisticsFor(ctx, record, actor.CanDatabase(record, permissions.DataRead)); err == nil {
			shown.Storage = readableSize(statistics.StorageBytes)
			shown.MemoryUsed = readableSize(statistics.MemoryBytes)
			shown.Connections = readableCount(statistics.Connections)
			if statistics.Keys >= 0 {
				shown.Tables = readableCount(statistics.Keys)
			}
			shown.Online = true
		}
		if actor.Can(permissions.RedisCredentialsManage) && actor.Can(permissions.RedisDataRead) && actor.Can(permissions.RedisDataWrite) {
			credentials, err := redisconnections.CurrentPrimary(ctx, record.Name)
			if err != nil {
				if err.Code == fiber.StatusForbidden {
					return err
				}
				shown.PasswordProblem = "The current password is unavailable. Rotate it to generate a new password without changing this database's data. Existing passwords are never reset when you open this page."
			} else {
				shown.RedisCredentials = credentials
				shown.Username = credentials.Username
				shown.URL = credentials.URL
			}
		}
	}
	return nil
}
