package redishost

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"sqldash/config"
	"sqldash/database"
	"sqldash/models"
)

func controlFailure(err error) string {
	switch {
	case errors.Is(err, IdentityMismatch):
		return "identity"
	case errors.Is(err, CapacityExceeded):
		return "capacity"
	case errors.Is(err, StorageUnowned), errors.Is(err, UnsafePath):
		return "storage"
	case errors.Is(err, PolicyUnconfirmed):
		return "policy"
	default:
		return "unavailable"
	}
}

func (self *manager) handleControl(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "application/json")
	response.Header().Set("Cache-Control", "no-store")
	token, err := controlToken()
	if err != nil || subtle.ConstantTimeCompare(token, []byte(request.Header.Get("X-Sqldash-Redis"))) != 1 || request.Method != http.MethodPost {
		clear(token)
		response.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(response).Encode(controlResponse{Failure: "unavailable"})
		return
	}
	clear(token)
	var asked controlRequest
	decoder := json.NewDecoder(io.LimitReader(request.Body, maximumControlBytes))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&asked) != nil {
		response.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(response).Encode(controlResponse{Failure: "unavailable"})
		return
	}
	if !self.ready.Load() {
		response.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(response).Encode(controlResponse{Failure: "unavailable"})
		return
	}
	if asked.Action == "ready" {
		_ = json.NewEncoder(response).Encode(controlResponse{})
		return
	}
	var record models.Database
	if asked.DatabaseID == 0 || database.DB.WithContext(request.Context()).First(&record, asked.DatabaseID).Error != nil || record.Kind() != models.EngineRedis || asked.Identity != "" && asked.Identity != record.ClusterID {
		response.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(response).Encode(controlResponse{Failure: "identity"})
		return
	}
	if asked.Action == "create" || asked.Action == "restart" {
		if err := ReconcileSettings(request.Context(), &record); err != nil {
			response.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(response).Encode(controlResponse{Failure: controlFailure(err)})
			return
		}
	}
	operation, lockError := lockOwnedFile(transferLockPath(record.ID), true)
	if lockError != nil {
		response.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(response).Encode(controlResponse{Failure: "policy"})
		return
	}
	defer operation.Close()
	if (asked.Action == "create" || asked.Action == "restart") && restorePending(request.Context(), &record) {
		response.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(response).Encode(controlResponse{Failure: "policy"})
		return
	}
	switch asked.Action {
	case "restore":
		err = self.restore(request.Context(), &record, asked.Generation, asked.RestoreToken)
	case "create":
		err = self.start(request.Context(), &record)
	case "stop":
		err = self.stop(&record, true)
	case "restart":
		if err = self.stop(&record, false); err == nil {
			err = self.start(request.Context(), &record)
		}
	case "delete":
		if err = self.remove(request.Context(), &record); err == nil {
			err = removeRestoreState(&record)
		}
	default:
		err = Unavailable
	}
	if err != nil {
		response.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(response).Encode(controlResponse{Failure: controlFailure(err)})
		return
	}
	result := controlResponse{Identity: record.ClusterID}
	if asked.Action != "delete" {
		if instance, err := Instance(request.Context(), &record); err == nil {
			result.Generation = instance.Generation
		}
	}
	_ = json.NewEncoder(response).Encode(result)
}

func instanceDirectoriesAbsent(databaseID uint) bool {
	for _, path := range []string{dataDirectory(databaseID), socketDirectory(databaseID), configurationDirectory(databaseID)} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			return false
		}
	}
	return true
}

func instanceFilesAbsent(databaseID uint) bool {
	if !instanceDirectoriesAbsent(databaseID) {
		return false
	}
	for _, path := range []string{manifestPath(databaseID), aclStatePath(databaseID), aclStatePath(databaseID) + ".applied", settingsIntentPath(databaseID), quarantinePath(databaseID), filepath.Join(config.RedisDirectory(), "logs", instanceName(databaseID)+".log")} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			return false
		}
	}
	return true
}

func (self *manager) remove(ctx context.Context, record *models.Database) error {
	if record.State != models.DatabaseDeleting && record.State != models.DatabaseApplying {
		return IdentityMismatch
	}
	instance, err := Instance(ctx, record)
	if err != nil {
		var remaining int64
		if database.DB.WithContext(ctx).Model(&models.RedisInstance{}).Where("database_id = ?", record.ID).Count(&remaining).Error == nil && remaining == 0 && instanceFilesAbsent(record.ID) {
			return nil
		}
		return err
	}
	if err := self.stop(record, false); err != nil {
		if !errors.Is(err, os.ErrNotExist) || !instanceDirectoriesAbsent(record.ID) {
			return err
		}
	}
	policy, err := lockOwnedFile(stateLockPath(record.ID), false)
	if err != nil {
		return PolicyUnconfirmed
	}
	defer policy.Close()
	ownership, err := lockOwnedFile(lockPath(record.ID), true)
	if err != nil {
		return PolicyUnconfirmed
	}
	defer ownership.Close()
	if _, err := readManifest(instance); err != nil {
		if !errors.Is(err, os.ErrNotExist) || !instanceDirectoriesAbsent(record.ID) {
			return err
		}
	}
	userID, _, err := instanceCredentials(record.ID)
	if err != nil {
		return err
	}
	for _, directory := range []struct {
		path  string
		owner int
	}{{dataDirectory(record.ID), userID}, {socketDirectory(record.ID), userID}, {configurationDirectory(record.ID), 0}} {
		if err := rejectSymlinks(directory.path); err != nil {
			return err
		}
		information, err := os.Lstat(directory.path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil || !information.IsDir() || !verifyOwner(information, directory.owner) {
			return UnsafePath
		}
		if err := os.RemoveAll(directory.path); err != nil {
			return Unavailable
		}
	}
	store, err := existingStore()
	if err != nil {
		return err
	}
	for _, profile := range systemProfiles {
		if err := store.Delete(secretNamespace(record.ID, instance.Identity), string(profile)); err != nil {
			return Unavailable
		}
	}
	if err := store.Delete(secretNamespace(record.ID, instance.Identity), string(referenceSecret)); err != nil {
		return Unavailable
	}
	for _, path := range []string{manifestPath(record.ID), aclStatePath(record.ID), aclStatePath(record.ID) + ".applied", settingsIntentPath(record.ID), quarantinePath(record.ID), filepath.Join(config.RedisDirectory(), "logs", instanceName(record.ID)+".log")} {
		if _, err := readOwnedFile(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return Unavailable
		}
	}
	if err := database.DB.Unscoped().Where("id = ? AND identity = ?", instance.ID, instance.Identity).Delete(&models.RedisInstance{}).Error; err != nil {
		return Unavailable
	}
	return nil
}
