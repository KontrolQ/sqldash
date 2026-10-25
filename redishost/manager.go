package redishost

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"time"

	"sqldash/config"
	"sqldash/database"
	"sqldash/models"
)

func Run(ctx context.Context, reconcile func(context.Context, *models.Database) error) (result error) {
	if err := verifyRuntime(ctx); err != nil {
		return err
	}
	if err := prepareRoot(); err != nil {
		return err
	}
	leader, err := lockOwnedFile(config.RedisRuntimePath()+string(os.PathSeparator)+"controller.lock", true)
	if err != nil {
		return Unavailable
	}
	defer leader.Close()
	if information, err := os.Lstat(controlSocketPath()); err == nil {
		if information.Mode()&os.ModeSocket == 0 || !verifyOwner(information, 0) {
			return UnsafePath
		}
		if err := os.Remove(controlSocketPath()); err != nil {
			return Unavailable
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return UnsafePath
	}
	listener, err := net.Listen("unix", controlSocketPath())
	if err != nil {
		return Unavailable
	}
	defer listener.Close()
	defer os.Remove(controlSocketPath())
	if err := os.Chmod(controlSocketPath(), 0o600); err != nil {
		return Unavailable
	}
	self := &manager{processes: make(map[uint]*managedInstance), reconcile: reconcile}
	mux := http.NewServeMux()
	mux.HandleFunc("/control", self.handleControl)
	self.server = &http.Server{Handler: mux, ReadHeaderTimeout: controlTimeout, ReadTimeout: controlTimeout, IdleTimeout: controlTimeout, MaxHeaderBytes: maximumControlBytes}
	served := make(chan error, 1)
	go func() { served <- self.server.Serve(listener) }()
	defer func() { result = errors.Join(result, self.shutdown()) }()
	self.reconcileInstances(ctx)
	self.ready.Store(true)
	ticker := time.NewTicker(config.Redis.RevalidationInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-served:
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			return Unavailable
		case <-ticker.C:
			if _, err := existingStore(); err != nil {
				return Unavailable
			}
			self.reconcileInstances(ctx)
		}
	}
}

func (self *manager) reconcileInstances(ctx context.Context) {
	self.mutex.Lock()
	for _, current := range self.processes {
		select {
		case <-current.finished:
			if current.control != nil {
				current.control.Close()
			}
			current.lock.Close()
			delete(self.processes, current.record.ID)
			_ = database.DB.WithContext(ctx).Model(&models.RedisInstance{}).Where("id = ? AND identity = ? AND state <> ?", current.instance.ID, current.instance.Identity, "blocked").Updates(map[string]any{"state": models.DatabaseFailed, "failure": "Redis instance stopped unexpectedly", "process_id": 0}).Error
		default:
		}
	}
	self.mutex.Unlock()
	var records []models.Database
	if err := database.DB.WithContext(ctx).Where("engine = ? AND state = ?", models.EngineRedis, models.DatabaseReady).Order("id asc").Find(&records).Error; err != nil {
		return
	}
	for index := range records {
		record := &records[index]
		instance, err := Instance(ctx, record)
		if err != nil || instance.State == "blocked" || instance.State == "restoring" {
			continue
		}
		if err := checkQuarantine(instance); err != nil {
			continue
		}
		if err := ReconcileSettings(ctx, record); err != nil {
			continue
		}
		if err := self.start(ctx, record); err != nil {
			_ = database.DB.WithContext(ctx).Model(instance).Updates(map[string]any{"state": models.DatabaseFailed, "failure": "Redis instance could not start"}).Error
			continue
		}
		if self.reconcile != nil {
			policyContext, cancel := context.WithTimeout(ctx, controlTimeout)
			err := self.reconcile(policyContext, record)
			cancel()
			if err != nil {
				self.mutex.Lock()
				if current := self.processes[record.ID]; current != nil {
					_ = self.stopLocked(current)
				}
				self.mutex.Unlock()
				_ = database.DB.WithContext(ctx).Model(instance).Updates(map[string]any{"state": "blocked", "failure": "Redis access policy could not be confirmed"}).Error
			}
		}
	}
}

func (self *manager) stop(record *models.Database, blocked bool) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	instance, err := Instance(context.Background(), record)
	if err != nil {
		return err
	}
	if current := self.processes[record.ID]; current != nil {
		if current.instance.Identity != instance.Identity {
			return IdentityMismatch
		}
		if err := self.stopLocked(current); err != nil {
			return err
		}
	} else {
		stored, err := readManifest(instance)
		if err != nil {
			return err
		}
		if stored.ProcessID > 0 && stored.ProcessStart != "" && processStartIdentity(stored.ProcessID) == stored.ProcessStart {
			return PolicyUnconfirmed
		}
		lock, err := lockOwnedFile(lockPath(record.ID), true)
		if err != nil {
			return PolicyUnconfirmed
		}
		lock.Close()
	}
	if blocked {
		return database.DB.Model(instance).Updates(map[string]any{"state": "blocked", "failure": "Redis instance is stopped pending recovery"}).Error
	}
	return nil
}
