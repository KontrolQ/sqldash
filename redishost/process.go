package redishost

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"sqldash/config"
	"sqldash/database"
	"sqldash/models"
)

type boundedLog struct {
	mutex sync.Mutex
	file  *os.File
	size  int64
}

func (self *boundedLog) Write(contents []byte) (int, error) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if self.size+int64(len(contents)) > maximumManifestBytes {
		if err := self.file.Truncate(0); err != nil {
			return 0, err
		}
		if _, err := self.file.Seek(0, io.SeekStart); err != nil {
			return 0, err
		}
		self.size = 0
	}
	written, err := self.file.Write(contents)
	self.size += int64(written)
	return written, err
}

func processLog(databaseID uint) (*boundedLog, error) {
	path := filepath.Join(config.RedisDirectory(), "logs", instanceName(databaseID)+".log")
	if err := rejectSymlinks(path); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, Unavailable
	}
	information, err := file.Stat()
	if err != nil || !information.Mode().IsRegular() || !verifyOwner(information, 0) || information.Mode().Perm()&0o077 != 0 {
		file.Close()
		return nil, UnsafePath
	}
	return &boundedLog{file: file, size: information.Size()}, nil
}

func prepareInstanceFiles(instance *models.RedisInstance) error {
	userID, groupID, err := instanceCredentials(instance.DatabaseID)
	if err != nil || len(socketPath(instance.DatabaseID)) > 100 {
		return Unavailable
	}
	if _, err := readManifest(instance); errors.Is(err, os.ErrNotExist) {
		if instance.State != models.DatabasePending {
			return StorageUnowned
		}
		for _, path := range []string{configurationDirectory(instance.DatabaseID), dataDirectory(instance.DatabaseID), socketDirectory(instance.DatabaseID)} {
			if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
				return StorageUnowned
			}
		}
		if err := writeManifest(instance); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if instance.Published || instance.State != models.DatabasePending {
		if err := verifyInstanceStorage(instance); err != nil {
			return err
		}
	}
	if err := ownedDirectory(configurationDirectory(instance.DatabaseID), 0o750, 0, groupID); err != nil {
		return err
	}
	if err := ownedDirectory(dataDirectory(instance.DatabaseID), 0o700, userID, groupID); err != nil {
		return err
	}
	if err := ownedDirectory(socketDirectory(instance.DatabaseID), 0o700, userID, groupID); err != nil {
		return err
	}
	if err := prepareACL(instance); err != nil {
		return err
	}
	return writeConfiguration(instance)
}

func (self *manager) start(ctx context.Context, record *models.Database) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if self.stopping.Load() || ctx.Err() != nil {
		return Unavailable
	}
	if current := self.processes[record.ID]; current != nil {
		select {
		case <-current.finished:
			current.lock.Close()
			current.control.Close()
			delete(self.processes, record.ID)
		default:
			if current.instance.Identity != record.ClusterID {
				return IdentityMismatch
			}
			return nil
		}
	}
	instance, err := prepareInstance(ctx, record)
	if err != nil {
		return err
	}
	if !validSettings(instance) {
		return InvalidSettings
	}
	if err := checkQuarantine(instance); err != nil {
		return err
	}
	if err := checkCapacity(database.DB.WithContext(ctx)); err != nil {
		return err
	}
	lock, err := lockOwnedFile(lockPath(record.ID), true)
	if err != nil {
		return Unavailable
	}
	owned := false
	defer func() {
		if !owned {
			lock.Close()
		}
	}()
	if err := recoverManifest(ctx, instance); err != nil {
		return err
	}
	if err := prepareInstanceFiles(instance); err != nil {
		return err
	}
	if stored, err := readManifest(instance); err != nil {
		return err
	} else if stored.ProcessID > 0 && stored.ProcessStart != "" && processStartIdentity(stored.ProcessID) == stored.ProcessStart {
		return Unavailable
	}
	if information, err := os.Lstat(socketPath(record.ID)); err == nil {
		userID, _, _ := instanceCredentials(record.ID)
		if information.Mode()&os.ModeSocket == 0 || !verifyOwner(information, userID) {
			return UnsafePath
		}
		if err := os.Remove(socketPath(record.ID)); err != nil {
			return Unavailable
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return UnsafePath
	}
	instance.Generation++
	instance.State = models.DatabaseApplying
	instance.ProcessID = 0
	if err := database.DB.WithContext(ctx).Model(instance).Updates(map[string]any{"generation": instance.Generation, "state": instance.State, "process_id": 0, "failure": ""}).Error; err != nil {
		return Unavailable
	}
	if err := writeManifest(instance); err != nil {
		return err
	}
	log, err := processLog(record.ID)
	if err != nil {
		return err
	}
	command := exec.Command(config.Redis.BinaryPath, configurationPath(record.ID))
	command.Dir = dataDirectory(record.ID)
	command.Env = []string{"PATH=/usr/local/bin:/usr/bin:/bin", "LANG=C.UTF-8", "HOME=/nonexistent"}
	command.Stdout = log
	command.Stderr = log
	if err := configureProcess(command, record.ID, lock); err != nil {
		log.file.Close()
		return err
	}
	if err := checkQuarantine(instance); err != nil {
		log.file.Close()
		return err
	}
	if err := command.Start(); err != nil {
		log.file.Close()
		return Unavailable
	}
	instance.ProcessID = command.Process.Pid
	if err := writeManifest(instance); err != nil {
		command.Process.Kill()
		command.Wait()
		log.file.Close()
		return err
	}
	current := &managedInstance{record: *record, instance: *instance, command: command, finished: make(chan error, 1), lock: lock}
	owned = true
	self.processes[record.ID] = current
	go func() {
		err := command.Wait()
		log.file.Close()
		current.finished <- err
		close(current.finished)
	}()
	startup, cancel := context.WithTimeout(ctx, config.Redis.StartupTimeout)
	defer cancel()
	for {
		control, connectError := privateClient(startup, record, instance, systemProfile)
		if connectError == nil {
			information, err := control.Info(startup, "server").Result()
			fields := informationFields(information)
			processID, parseError := strconv.Atoi(fields["process_id"])
			if err != nil || parseError != nil || processID != instance.ProcessID || !strings.HasPrefix(fields["redis_version"], "8.") || fields["run_id"] == "" {
				control.Close()
				self.stopLocked(current)
				return IdentityMismatch
			}
			for _, name := range []string{"JSON.GET", "BF.ADD", "FT.SEARCH", "TS.ADD", "VADD"} {
				capabilities, err := control.Do(startup, "COMMAND", "INFO", name).Slice()
				if err != nil || len(capabilities) != 1 || capabilities[0] == nil {
					control.Close()
					self.stopLocked(current)
					return Unavailable
				}
			}
			if err := checkQuarantine(instance); err != nil {
				control.Close()
				self.stopLocked(current)
				return err
			}
			current.control = control
			instance.State = models.DatabaseReady
			instance.RunID = fields["run_id"]
			now := time.Now()
			instance.ReadyAt = &now
			current.instance = *instance
			if err := database.DB.WithContext(ctx).Model(instance).Updates(map[string]any{"state": instance.State, "process_id": instance.ProcessID, "run_id": instance.RunID, "ready_at": now, "failure": ""}).Error; err != nil {
				self.stopLocked(current)
				return Unavailable
			}
			return nil
		}
		select {
		case <-startup.Done():
			self.stopLocked(current)
			return Unavailable
		case <-current.finished:
			self.stopLocked(current)
			return Unavailable
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (self *manager) stopLocked(current *managedInstance) error {
	err := stopProcess(current)
	if !errors.Is(err, PolicyUnconfirmed) {
		delete(self.processes, current.record.ID)
	}
	return err
}

func stopProcess(current *managedInstance) error {
	if current.control != nil {
		current.control.Close()
	}
	_ = current.command.Process.Signal(syscall.SIGTERM)
	forced := false
	select {
	case <-current.finished:
	case <-time.After(config.Redis.ShutdownTimeout):
		_ = current.command.Process.Kill()
		select {
		case <-current.finished:
			forced = true
		case <-time.After(5 * time.Second):
			return PolicyUnconfirmed
		}
	}
	current.lock.Close()
	current.instance.State = "stopped"
	current.instance.ProcessID = 0
	now := time.Now()
	failure := ""
	if forced {
		failure = "Redis required forced shutdown"
	}
	if err := database.DB.Model(&models.RedisInstance{}).Where("id = ? AND identity = ?", current.instance.ID, current.instance.Identity).Updates(map[string]any{"state": "stopped", "process_id": 0, "stopped_at": now, "failure": failure}).Error; err != nil {
		return Unavailable
	}
	if err := writeManifest(&current.instance); err != nil {
		return err
	}
	return nil
}
