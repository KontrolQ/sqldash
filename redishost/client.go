package redishost

import (
	"context"
	"net"
	"os"
	"strconv"
	"strings"

	"sqldash/config"
	"sqldash/models"
	"sqldash/rediswire"

	redisclient "github.com/redis/go-redis/v9"
)

func Dial(ctx context.Context, record *models.Database) (net.Conn, error) {
	instance, err := Instance(ctx, record)
	if err != nil {
		return nil, err
	}
	return dialInstance(ctx, instance)
}

func dialInstance(ctx context.Context, instance *models.RedisInstance) (net.Conn, error) {
	if !managedSupported() {
		return nil, Unavailable
	}
	if _, err := readManifest(instance); err != nil {
		return nil, err
	}
	if err := checkQuarantine(instance); err != nil {
		return nil, err
	}
	path := socketPath(instance.DatabaseID)
	if err := rejectSymlinks(path); err != nil {
		return nil, err
	}
	information, err := os.Lstat(path)
	userID, _, identityError := instanceCredentials(instance.DatabaseID)
	if err != nil || identityError != nil || information.Mode()&os.ModeSocket == 0 || !verifyOwner(information, userID) || information.Mode().Perm() != 0o600 {
		return nil, Unavailable
	}
	connection, err := (&net.Dialer{Timeout: controlTimeout}).DialContext(ctx, "unix", path)
	if err != nil {
		return nil, Unavailable
	}
	if !socketPeerMatches(connection, instance.DatabaseID, instance.ProcessID) {
		connection.Close()
		return nil, IdentityMismatch
	}
	if _, err := readManifest(instance); err != nil {
		connection.Close()
		return nil, err
	}
	return connection, nil
}

func Client(ctx context.Context, record *models.Database, profile Profile) (*redisclient.Client, error) {
	if profile == systemProfile {
		return nil, InvalidCredential
	}
	if _, err := profileRules(profile, false, false); err != nil {
		return nil, err
	}
	instance, err := Instance(ctx, record)
	if err != nil {
		return nil, err
	}
	if instance.State != models.DatabaseReady {
		return nil, Unavailable
	}
	return privateClient(ctx, record, instance, profile)
}

func privateClient(ctx context.Context, record *models.Database, instance *models.RedisInstance, profile Profile) (*redisclient.Client, error) {
	if record.ID != instance.DatabaseID || record.ClusterID != instance.Identity {
		return nil, IdentityMismatch
	}
	password, err := systemPassword(instance, profile, false)
	if err != nil {
		return nil, err
	}
	client := redisclient.NewClient(&redisclient.Options{
		Addr: socketPath(record.ID), Network: "unix", Protocol: 2,
		Username: systemUsername(record.ID, profile), Password: string(password),
		DisableIdentity: true, MaxRetries: -1, PoolSize: 2, MaxIdleConns: 1,
		DialTimeout: controlTimeout, ReadTimeout: config.Redis.CommandTimeout, WriteTimeout: controlTimeout,
		ContextTimeoutEnabled: true,
		Dialer: func(parent context.Context, network, address string) (net.Conn, error) {
			connection, err := dialInstance(parent, instance)
			if err != nil {
				return nil, err
			}
			return rediswire.Guard(connection, rediswire.Limits{MaximumBytes: config.Redis.MaximumValueBytes + 1024*1024, MaximumBulkBytes: config.Redis.MaximumValueBytes, MaximumElements: 32768, MaximumDepth: 64}), nil
		},
	})
	clear(password)
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, Unavailable
	}
	return client, nil
}

func informationFields(text string) map[string]string {
	fields := make(map[string]string)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, present := strings.Cut(line, ":")
		if present {
			fields[key] = value
		}
	}
	return fields
}

func KillClientForRun(ctx context.Context, record *models.Database, clientID int64, runID string) error {
	if clientID < 1 || runID == "" {
		return InvalidCredential
	}
	instance, err := Instance(ctx, record)
	if err != nil {
		return err
	}
	control, err := privateClient(ctx, record, instance, systemProfile)
	if err != nil {
		return PolicyUnconfirmed
	}
	defer control.Close()
	information, err := control.Info(ctx, "server").Result()
	if err != nil {
		return PolicyUnconfirmed
	}
	if informationFields(information)["run_id"] != runID {
		return nil
	}
	if err := control.ClientKillByFilter(ctx, "ID", strconv.FormatInt(clientID, 10)).Err(); err != nil {
		return PolicyUnconfirmed
	}
	clients, err := control.ClientList(ctx).Result()
	if err != nil {
		return PolicyUnconfirmed
	}
	for _, line := range strings.Split(clients, "\n") {
		for _, field := range strings.Fields(line) {
			if field == "id="+strconv.FormatInt(clientID, 10) {
				return PolicyUnconfirmed
			}
		}
	}
	return nil
}
