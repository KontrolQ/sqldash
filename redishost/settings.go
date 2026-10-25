package redishost

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"sqldash/config"
	"sqldash/database"
	"sqldash/models"

	redisclient "github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Settings struct {
	MemoryLimit    int64
	EvictionPolicy string
	Persistence    string
	Fsync          string
}

type settingsIntent struct {
	DatabaseID          uint
	Identity            string
	Generation          uint64
	ProcessID           int
	ProcessStart        string
	RunID               string
	Before              Settings
	After               Settings
	Phase               string
	ConfigurationDigest string
}

const (
	settingsTimeout      = 20 * time.Second
	settingsPause        = 60 * time.Second
	settingsSaveSchedule = "900 1 300 10 60 10000"
)

func settingsIntentPath(databaseID uint) string {
	return filepath.Join(stateDirectory(), instanceName(databaseID)+".settings.json")
}

func instanceSettings(instance *models.RedisInstance) Settings {
	return Settings{MemoryLimit: instance.MemoryLimit, EvictionPolicy: instance.EvictionPolicy, Persistence: instance.Persistence, Fsync: instance.Fsync}
}

func settingsInstance(instance *models.RedisInstance, settings Settings) *models.RedisInstance {
	candidate := *instance
	candidate.MemoryLimit = settings.MemoryLimit
	candidate.EvictionPolicy = settings.EvictionPolicy
	candidate.Persistence = settings.Persistence
	candidate.Fsync = settings.Fsync
	return &candidate
}

func writeSettingsIntent(intent *settingsIntent) error {
	contents, err := json.Marshal(intent)
	if err != nil {
		return Unavailable
	}
	return ownedFile(settingsIntentPath(intent.DatabaseID), contents, 0o600, 0)
}

func readSettingsIntent(databaseID uint) (*settingsIntent, error) {
	contents, err := readOwnedFile(settingsIntentPath(databaseID))
	if err != nil {
		return nil, err
	}
	var intent settingsIntent
	if json.Unmarshal(contents, &intent) != nil || intent.DatabaseID != databaseID || !validIdentity(intent.Identity) || intent.Generation == 0 || intent.ProcessID < 1 || intent.ProcessStart == "" || intent.RunID == "" {
		return nil, IdentityMismatch
	}
	if intent.Phase != "prepared" && intent.Phase != "snapshot" && intent.Phase != "durable" {
		return nil, IdentityMismatch
	}
	if !validSettings(settingsInstance(&models.RedisInstance{}, intent.Before)) || !validSettings(settingsInstance(&models.RedisInstance{}, intent.After)) {
		return nil, InvalidSettings
	}
	return &intent, nil
}

func matchesSettingsIntent(instance *models.RedisInstance, intent *settingsIntent) bool {
	return intent.Identity == instance.Identity && intent.Generation == instance.Generation && intent.ProcessID == instance.ProcessID && intent.RunID == instance.RunID
}

func removeSettingsIntent(databaseID uint) error {
	if _, err := readOwnedFile(settingsIntentPath(databaseID)); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	if err := os.Remove(settingsIntentPath(databaseID)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Unavailable
	}
	return syncRestoreDirectory(stateDirectory())
}

func settingsConfigurationDigest(instance *models.RedisInstance) (string, error) {
	path := configurationPath(instance.DatabaseID)
	if err := rejectSymlinks(path); err != nil {
		return "", err
	}
	information, err := os.Lstat(path)
	if err != nil || !information.Mode().IsRegular() || !verifyOwner(information, 0) || information.Mode().Perm() != 0o640 || information.Size() > 64*1024 {
		return "", UnsafePath
	}
	file, err := os.Open(path)
	if err != nil {
		return "", Unavailable
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(information, opened) {
		return "", UnsafePath
	}
	contents, err := io.ReadAll(io.LimitReader(file, 64*1024+1))
	if err != nil || len(contents) > 64*1024 {
		return "", Unavailable
	}
	digest := sha256.Sum256(contents)
	return hex.EncodeToString(digest[:]), nil
}

func settingsIdentity(ctx context.Context, record *models.Database, instance *models.RedisInstance, client *redisclient.Client) error {
	current, err := Instance(ctx, record)
	if err != nil {
		return err
	}
	manifest, err := readManifest(current)
	if err != nil {
		return err
	}
	if current.Identity != instance.Identity || current.Generation != instance.Generation || current.ProcessID != instance.ProcessID || current.RunID != instance.RunID || current.State != models.DatabaseReady && current.State != models.DatabaseApplying || manifest.ProcessID != instance.ProcessID || manifest.ProcessStart == "" || processStartIdentity(instance.ProcessID) != manifest.ProcessStart {
		return IdentityMismatch
	}
	if err := checkQuarantine(current); err != nil {
		return err
	}
	information, err := client.Info(ctx, "server").Result()
	if err != nil {
		return Unavailable
	}
	fields := informationFields(information)
	if fields["run_id"] != instance.RunID || fields["process_id"] != strconv.Itoa(instance.ProcessID) {
		return IdentityMismatch
	}
	return nil
}

func settingsConfiguration(ctx context.Context, client *redisclient.Client, names ...string) (map[string]string, error) {
	arguments := []any{"CONFIG", "GET"}
	for _, name := range names {
		arguments = append(arguments, name)
	}
	values, err := client.Do(ctx, arguments...).StringSlice()
	if err != nil || len(values) != len(names)*2 {
		return nil, Unavailable
	}
	configuration := make(map[string]string, len(names))
	for index := 0; index < len(values); index += 2 {
		configuration[values[index]] = values[index+1]
	}
	return configuration, nil
}

func runtimeSettings(ctx context.Context, client *redisclient.Client) (Settings, error) {
	values, err := settingsConfiguration(ctx, client, "maxmemory", "maxmemory-policy", "appendonly", "appendfsync", "save")
	if err != nil {
		return Settings{}, Unavailable
	}
	memory, err := strconv.ParseInt(values["maxmemory"], 10, 64)
	if err != nil {
		return Settings{}, Unavailable
	}
	persistence := "aof-rdb"
	if values["appendonly"] == "no" {
		persistence = "rdb"
	} else if values["appendonly"] != "yes" {
		return Settings{}, Unavailable
	}
	schedule := strings.Join(strings.Fields(values["save"]), " ")
	if schedule == "" {
		if persistence == "rdb" {
			return Settings{}, PolicyUnconfirmed
		}
		persistence = "aof"
	} else if schedule != settingsSaveSchedule {
		return Settings{}, PolicyUnconfirmed
	}
	return Settings{MemoryLimit: memory, EvictionPolicy: values["maxmemory-policy"], Persistence: persistence, Fsync: values["appendfsync"]}, nil
}

func validateSettingsMemory(ctx context.Context, client *redisclient.Client, limit int64) error {
	information, err := client.Info(ctx, "memory").Result()
	if err != nil {
		return Unavailable
	}
	used, err := strconv.ParseInt(informationFields(information)["used_memory"], 10, 64)
	if err != nil || used < 0 {
		return Unavailable
	}
	if limit < used || limit-used < 1024*1024 {
		return CapacityExceeded
	}
	return nil
}

func reservedSettings(before, after Settings) Settings {
	after.MemoryLimit = max(before.MemoryLimit, after.MemoryLimit)
	return after
}

func reserveSettings(ctx context.Context, instance *models.RedisInstance, settings Settings) error {
	settings = reservedSettings(instanceSettings(instance), settings)
	return database.DB.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		var reserved int64
		if err := transaction.Model(&models.RedisInstance{}).Select("coalesce(sum(memory_limit + ?), 0)", instanceOverhead).Scan(&reserved).Error; err != nil {
			return Unavailable
		}
		if reserved < instance.MemoryLimit+instanceOverhead || settings.MemoryLimit > config.Redis.MemoryBudget || reserved-instance.MemoryLimit > config.Redis.MemoryBudget-settings.MemoryLimit {
			return CapacityExceeded
		}
		updated := transaction.Model(&models.RedisInstance{}).Where("id = ? AND identity = ? AND generation = ? AND process_id = ? AND run_id = ? AND state = ? AND memory_limit = ? AND eviction_policy = ? AND persistence = ? AND fsync = ?", instance.ID, instance.Identity, instance.Generation, instance.ProcessID, instance.RunID, models.DatabaseReady, instance.MemoryLimit, instance.EvictionPolicy, instance.Persistence, instance.Fsync).Updates(map[string]any{"memory_limit": settings.MemoryLimit, "eviction_policy": settings.EvictionPolicy, "persistence": settings.Persistence, "fsync": settings.Fsync, "state": models.DatabaseApplying})
		if updated.Error != nil || updated.RowsAffected != 1 {
			return PolicyUnconfirmed
		}
		return nil
	})
}

func syncSettingsMetadata(ctx context.Context) error {
	// NORMAL WAL commits must be checkpointed before their recovery intent disappears.
	var busy, frames, completed int64
	if err := database.DB.WithContext(ctx).Raw("PRAGMA wal_checkpoint(FULL)").Row().Scan(&busy, &frames, &completed); err != nil || busy != 0 || frames < 0 || completed != frames {
		return PolicyUnconfirmed
	}
	return nil
}

func persistSettings(ctx context.Context, instance *models.RedisInstance, expected Settings, wanted Settings, state string) error {
	updated := database.DB.WithContext(ctx).Model(&models.RedisInstance{}).Where("id = ? AND identity = ? AND generation = ? AND process_id = ? AND run_id = ? AND memory_limit = ? AND eviction_policy = ? AND persistence = ? AND fsync = ? AND state IN ?", instance.ID, instance.Identity, instance.Generation, instance.ProcessID, instance.RunID, expected.MemoryLimit, expected.EvictionPolicy, expected.Persistence, expected.Fsync, []string{models.DatabaseApplying, models.DatabaseReady}).Updates(map[string]any{"memory_limit": wanted.MemoryLimit, "eviction_policy": wanted.EvictionPolicy, "persistence": wanted.Persistence, "fsync": wanted.Fsync, "state": state})
	if updated.Error != nil || updated.RowsAffected != 1 {
		return PolicyUnconfirmed
	}
	return nil
}

func settingsSnapshot(ctx context.Context, record *models.Database, instance *models.RedisInstance, client *redisclient.Client) error {
	before, err := beginSnapshot(ctx, client, func(current context.Context) error { return settingsIdentity(current, record, instance, client) })
	if err != nil {
		return err
	}
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := settingsIdentity(ctx, record, instance, client); err != nil {
			return err
		}
		information, err := client.Info(ctx, "persistence").Result()
		if err != nil {
			return Unavailable
		}
		fields := informationFields(information)
		if fields["rdb_bgsave_in_progress"] == "0" {
			completed, err := strconv.ParseUint(fields["rdb_saves"], 10, 64)
			if err != nil || completed != before+1 || fields["rdb_last_bgsave_status"] != "ok" {
				return Unavailable
			}
			userID, _, err := instanceCredentials(record.ID)
			if err != nil {
				return err
			}
			root, err := openSnapshotRoot(dataDirectory(record.ID), userID)
			if err != nil {
				return err
			}
			defer root.Close()
			file, err := openSnapshotFile(root, "dump.rdb", userID)
			if err != nil {
				return err
			}
			return file.Close()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func waitSettingsAOF(ctx context.Context, record *models.Database, instance *models.RedisInstance, client *redisclient.Client, previous uint64) error {
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := settingsIdentity(ctx, record, instance, client); err != nil {
			return err
		}
		information, err := client.Info(ctx, "persistence").Result()
		if err != nil {
			return Unavailable
		}
		fields := informationFields(information)
		completed, err := strconv.ParseUint(fields["aof_rewrites"], 10, 64)
		if err != nil || fields["aof_enabled"] != "1" || fields["aof_last_write_status"] != "ok" {
			return Unavailable
		}
		if fields["aof_rewrite_in_progress"] == "0" && fields["aof_rewrite_scheduled"] == "0" {
			if completed <= previous || fields["aof_last_bgrewrite_status"] != "ok" {
				return Unavailable
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func applyRuntimeSettings(ctx context.Context, record *models.Database, instance *models.RedisInstance, client *redisclient.Client, wanted Settings) error {
	current, readError := runtimeSettings(ctx, client)
	if readError == nil && current == wanted {
		return nil
	}
	configuration, err := client.ConfigGet(ctx, "appendonly").Result()
	if err != nil {
		return Unavailable
	}
	appendOnly := configuration["appendonly"]
	if appendOnly != "yes" && appendOnly != "no" {
		return PolicyUnconfirmed
	}
	if err := validateSettingsMemory(ctx, client, wanted.MemoryLimit); err != nil {
		return err
	}
	schedule := settingsSaveSchedule
	if wanted.Persistence == "aof" {
		schedule = ""
	}
	if err := client.Do(ctx, "CONFIG", "SET", "maxmemory", strconv.FormatInt(wanted.MemoryLimit, 10), "maxmemory-policy", wanted.EvictionPolicy, "appendfsync", "always", "save", schedule).Err(); err != nil {
		return Unavailable
	}
	if wanted.Persistence == "rdb" && appendOnly == "yes" {
		if err := client.ConfigSet(ctx, "appendonly", "no").Err(); err != nil {
			return Unavailable
		}
	} else if wanted.Persistence != "rdb" && appendOnly == "no" {
		information, err := client.Info(ctx, "persistence").Result()
		if err != nil {
			return Unavailable
		}
		previous, err := strconv.ParseUint(informationFields(information)["aof_rewrites"], 10, 64)
		if err != nil {
			return Unavailable
		}
		if err := client.ConfigSet(ctx, "appendonly", "yes").Err(); err != nil {
			return Unavailable
		}
		if err := waitSettingsAOF(ctx, record, instance, client, previous); err != nil {
			return err
		}
	}
	if err := client.ConfigSet(ctx, "appendfsync", wanted.Fsync).Err(); err != nil {
		return Unavailable
	}
	actual, err := runtimeSettings(ctx, client)
	if err != nil || actual != wanted {
		return PolicyUnconfirmed
	}
	return settingsIdentity(ctx, record, instance, client)
}

func quarantineSettings(ctx context.Context, record *models.Database, instance *models.RedisInstance) error {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	_ = Quarantine(cleanup, record, instance.Generation)
	return PolicyUnconfirmed
}

func rollbackSettings(ctx context.Context, record *models.Database, instance *models.RedisInstance, intent *settingsIntent) error {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), settingsTimeout)
	defer cancel()
	client, err := privateClient(cleanup, record, instance, systemProfile)
	if err != nil {
		return quarantineSettings(cleanup, record, instance)
	}
	defer client.Close()
	if err := settingsIdentity(cleanup, record, instance, client); err != nil {
		return quarantineSettings(cleanup, record, instance)
	}
	if err := client.Do(cleanup, "CLIENT", "PAUSE", settingsPause.Milliseconds(), "WRITE").Err(); err != nil {
		return quarantineSettings(cleanup, record, instance)
	}
	if err := applyRuntimeSettings(cleanup, record, instance, client, intent.Before); err != nil {
		return quarantineSettings(cleanup, record, instance)
	}
	if err := writeConfiguration(settingsInstance(instance, intent.Before)); err != nil {
		return quarantineSettings(cleanup, record, instance)
	}
	current, err := Instance(cleanup, record)
	if err != nil {
		return quarantineSettings(cleanup, record, instance)
	}
	expected := instanceSettings(current)
	if expected != reservedSettings(intent.Before, intent.After) && expected != intent.Before {
		return quarantineSettings(cleanup, record, instance)
	}
	if err := persistSettings(cleanup, instance, expected, intent.Before, models.DatabaseReady); err != nil {
		return quarantineSettings(cleanup, record, instance)
	}
	if err := syncSettingsMetadata(cleanup); err != nil {
		return quarantineSettings(cleanup, record, instance)
	}
	if err := removeSettingsIntent(record.ID); err != nil {
		return quarantineSettings(cleanup, record, instance)
	}
	if err := client.Do(cleanup, "CLIENT", "UNPAUSE").Err(); err != nil {
		return quarantineSettings(cleanup, record, instance)
	}
	return nil
}

func UpdateSettings(ctx context.Context, record *models.Database, asked Settings, authorization ...func(context.Context) error) (result error) {
	if len(authorization) > 1 || len(authorization) == 1 && authorization[0] == nil {
		return InvalidSettings
	}
	checkAuthority := func(parent context.Context) error {
		if len(authorization) == 0 {
			return parent.Err()
		}
		return authorization[0](parent)
	}
	if err := checkAuthority(ctx); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !validSettings(settingsInstance(&models.RedisInstance{}, asked)) {
		return InvalidSettings
	}
	stored, err := transferRecord(ctx, record, models.DatabaseReady)
	if err != nil {
		return err
	}
	lock, err := lockOwnedFile(transferLockPath(record.ID), true)
	if err != nil {
		return PolicyUnconfirmed
	}
	defer lock.Close()
	capacity, err := lockOwnedFile(filepath.Join(stateDirectory(), "settings.capacity.lock"), true)
	if err != nil {
		return PolicyUnconfirmed
	}
	defer capacity.Close()
	if _, err := readOwnedFile(settingsIntentPath(record.ID)); !errors.Is(err, os.ErrNotExist) {
		return PolicyUnconfirmed
	}
	instance, err := Instance(ctx, stored)
	if err != nil {
		return err
	}
	if instance.State != models.DatabaseReady {
		return PolicyUnconfirmed
	}
	client, err := privateClient(ctx, stored, instance, systemProfile)
	if err != nil {
		return err
	}
	defer client.Close()
	if err := settingsIdentity(ctx, stored, instance, client); err != nil {
		return err
	}
	before := instanceSettings(instance)
	actual, err := runtimeSettings(ctx, client)
	if err != nil || actual != before {
		return PolicyUnconfirmed
	}
	if asked == before {
		return nil
	}
	if err := validateSettingsMemory(ctx, client, asked.MemoryLimit); err != nil {
		return err
	}
	manifest, err := readManifest(instance)
	if err != nil {
		return err
	}
	intent := &settingsIntent{DatabaseID: record.ID, Identity: instance.Identity, Generation: instance.Generation, ProcessID: instance.ProcessID, ProcessStart: manifest.ProcessStart, RunID: instance.RunID, Before: before, After: asked, Phase: "prepared"}
	if err := writeSettingsIntent(intent); err != nil {
		return err
	}
	if err := reserveSettings(ctx, instance, asked); err != nil {
		_ = removeSettingsIntent(record.ID)
		return err
	}
	durable := false
	defer func() {
		if result != nil {
			if durable {
				result = quarantineSettings(ctx, stored, instance)
			} else if err := rollbackSettings(ctx, stored, instance, intent); err != nil {
				result = err
			}
		}
	}()
	operation, cancel := context.WithTimeout(ctx, settingsTimeout)
	defer cancel()
	if err := client.Do(operation, "CLIENT", "PAUSE", settingsPause.Milliseconds(), "WRITE").Err(); err != nil {
		return Unavailable
	}
	if err := settingsSnapshot(operation, stored, instance, client); err != nil {
		if operation.Err() != nil {
			return operation.Err()
		}
		return err
	}
	intent.Phase = "snapshot"
	if err := writeSettingsIntent(intent); err != nil {
		return err
	}
	if err := operation.Err(); err != nil {
		return err
	}
	if err := checkAuthority(operation); err != nil {
		return err
	}
	if err := applyRuntimeSettings(operation, stored, instance, client, asked); err != nil {
		if operation.Err() != nil {
			return operation.Err()
		}
		return err
	}
	if err := checkAuthority(operation); err != nil {
		return err
	}
	if err := writeConfiguration(settingsInstance(instance, asked)); err != nil {
		return err
	}
	intent.ConfigurationDigest, err = settingsConfigurationDigest(instance)
	if err != nil {
		return err
	}
	if err := checkAuthority(operation); err != nil {
		return err
	}
	intent.Phase = "durable"
	if err := writeSettingsIntent(intent); err != nil {
		return err
	}
	durable = true
	completion, complete := context.WithTimeout(context.WithoutCancel(ctx), controlTimeout)
	defer complete()
	if err := persistSettings(completion, instance, reservedSettings(before, asked), asked, models.DatabaseReady); err != nil {
		return err
	}
	if err := syncSettingsMetadata(completion); err != nil {
		return err
	}
	if err := removeSettingsIntent(record.ID); err != nil {
		return err
	}
	if err := client.Do(completion, "CLIENT", "UNPAUSE").Err(); err != nil {
		return err
	}
	return nil
}

func ReconcileSettings(ctx context.Context, record *models.Database) error {
	if record == nil || record.ID == 0 || record.Kind() != models.EngineRedis {
		return IdentityMismatch
	}
	intent, err := readSettingsIntent(record.ID)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	instance, err := Instance(ctx, record)
	if err != nil {
		return err
	}
	if !matchesSettingsIntent(instance, intent) {
		return PolicyUnconfirmed
	}
	lock, err := lockOwnedFile(transferLockPath(record.ID), true)
	if err != nil {
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) {
			return err
		}
		if processStartIdentity(instance.ProcessID) == intent.ProcessStart {
			return nil
		}
		return PolicyUnconfirmed
	}
	defer lock.Close()
	intent, err = readSettingsIntent(record.ID)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	instance, err = Instance(ctx, record)
	if err != nil {
		return err
	}
	if !matchesSettingsIntent(instance, intent) {
		return PolicyUnconfirmed
	}
	if intent.Phase != "durable" {
		if processStartIdentity(instance.ProcessID) == intent.ProcessStart {
			return rollbackSettings(ctx, record, instance, intent)
		}
		return quarantineSettings(ctx, record, instance)
	}
	currentSettings := instanceSettings(instance)
	if currentSettings != intent.After && currentSettings != reservedSettings(intent.Before, intent.After) {
		return quarantineSettings(ctx, record, instance)
	}
	digest, err := settingsConfigurationDigest(instance)
	if err != nil || digest != intent.ConfigurationDigest || !validIdentity(digest) {
		return quarantineSettings(ctx, record, instance)
	}
	var resumedClient *redisclient.Client
	if processStartIdentity(instance.ProcessID) == intent.ProcessStart {
		client, err := privateClient(ctx, record, instance, systemProfile)
		if err != nil {
			return quarantineSettings(ctx, record, instance)
		}
		defer client.Close()
		resumedClient = client
		if err := settingsIdentity(ctx, record, instance, client); err != nil {
			return quarantineSettings(ctx, record, instance)
		}
		actual, err := runtimeSettings(ctx, client)
		if err != nil || actual != intent.After {
			return quarantineSettings(ctx, record, instance)
		}
		if err := persistSettings(ctx, instance, currentSettings, intent.After, models.DatabaseReady); err != nil {
			return quarantineSettings(ctx, record, instance)
		}
	} else {
		if err := persistSettings(ctx, instance, currentSettings, intent.After, models.DatabaseReady); err != nil {
			return err
		}
	}
	if err := syncSettingsMetadata(ctx); err != nil {
		return quarantineSettings(ctx, record, instance)
	}
	if err := removeSettingsIntent(record.ID); err != nil {
		return quarantineSettings(ctx, record, instance)
	}
	if resumedClient != nil {
		if err := resumedClient.Do(ctx, "CLIENT", "UNPAUSE").Err(); err != nil {
			return quarantineSettings(ctx, record, instance)
		}
	}
	return nil
}
