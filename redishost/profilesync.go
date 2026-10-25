package redishost

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"

	"sqldash/models"

	redisclient "github.com/redis/go-redis/v9"
)

type appliedUser struct {
	Desired string
	Actual  string
}

type appliedACL struct {
	DatabaseID uint
	Identity   string
	Generation uint64
	Users      map[string]appliedUser
}

func aclFingerprint(value any) string {
	encoded, _ := json.Marshal(value)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func readAppliedACL(instance *models.RedisInstance) (*appliedACL, error) {
	contents, err := readOwnedFile(aclStatePath(instance.DatabaseID) + ".applied")
	if errors.Is(err, os.ErrNotExist) {
		return &appliedACL{DatabaseID: instance.DatabaseID, Identity: instance.Identity, Generation: instance.Generation, Users: map[string]appliedUser{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var applied appliedACL
	if json.Unmarshal(contents, &applied) != nil || applied.DatabaseID != instance.DatabaseID || !validIdentity(applied.Identity) || applied.Generation == 0 || applied.Users == nil || len(applied.Users) > 1032 {
		return nil, IdentityMismatch
	}
	if applied.Identity != instance.Identity || applied.Generation != instance.Generation {
		applied.Identity = instance.Identity
		applied.Generation = instance.Generation
		applied.Users = map[string]appliedUser{}
	}
	return &applied, nil
}

func writeAppliedACL(instance *models.RedisInstance, applied *appliedACL) error {
	contents, err := json.Marshal(applied)
	if err != nil || len(contents) > maximumManifestBytes {
		return PolicyUnconfirmed
	}
	if err := ownedFile(aclStatePath(instance.DatabaseID)+".applied", contents, 0o600, 0); err != nil {
		return PolicyUnconfirmed
	}
	return nil
}

func actualUser(ctx context.Context, control *redisclient.Client, username string) (any, error) {
	value, err := control.Do(ctx, "ACL", "GETUSER", username).Result()
	if errors.Is(err, redisclient.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, PolicyUnconfirmed
	}
	return value, nil
}

func confirmsVerifier(actual any, digest string) bool {
	properties, present := actual.([]any)
	if !present || len(properties)%2 != 0 {
		return false
	}
	enabled, passwordMatches := false, false
	for index := 0; index < len(properties); index += 2 {
		name, _ := properties[index].(string)
		values, _ := properties[index+1].([]any)
		switch name {
		case "flags":
			for _, value := range values {
				if value == "on" {
					enabled = true
				}
				if value == "nopass" {
					return false
				}
			}
		case "passwords":
			passwordMatches = len(values) == 1 && values[0] == digest
		}
	}
	return enabled && passwordMatches
}

func SetVerifier(ctx context.Context, record *models.Database, username, digest string, profile Profile, subscribe, publish bool) error {
	if !publicUsername(username) || !validPasswordHash(digest) || profile != ReadOnly && profile != ReadWrite {
		return InvalidCredential
	}
	digest = strings.ToLower(digest)
	instance, err := Instance(ctx, record)
	if err != nil {
		return err
	}
	lock, err := lockOwnedFile(stateLockPath(record.ID), false)
	if err != nil {
		return Unavailable
	}
	defer lock.Close()
	document, err := readACLDocument(instance)
	if err != nil {
		return err
	}
	if _, exists := document.Users[username]; !exists && len(document.Users) >= 1031 {
		return CapacityExceeded
	}
	applied, err := readAppliedACL(instance)
	if err != nil {
		return err
	}
	control, err := privateClient(ctx, record, instance, systemProfile)
	if err != nil {
		return PolicyUnconfirmed
	}
	defer control.Close()
	actual, err := actualUser(ctx, control, username)
	if err != nil {
		return err
	}
	desired := aclUser{Hash: digest, Profile: profile, Subscribe: subscribe, Publish: publish}
	rules, err := profileRules(profile, subscribe, publish)
	if err != nil {
		return err
	}
	desiredFingerprint := aclFingerprint([]any{desired, rules})
	previous, exists := document.Users[username]
	confirmation := applied.Users[username]
	if exists && previous == desired && confirmation.Desired == desiredFingerprint && confirmation.Actual == aclFingerprint(actual) && confirmsVerifier(actual, digest) {
		return nil
	}
	document.Users[username] = desired
	if err := writeACLDocument(instance, document); err != nil {
		return err
	}
	if err := disableAndDisconnect(ctx, control, username); err != nil {
		return err
	}
	if err := control.Do(ctx, "ACL", "LOAD").Err(); err != nil {
		return PolicyUnconfirmed
	}
	actual, err = actualUser(ctx, control, username)
	if err != nil || !confirmsVerifier(actual, digest) {
		return PolicyUnconfirmed
	}
	applied.Users[username] = appliedUser{Desired: desiredFingerprint, Actual: aclFingerprint(actual)}
	return writeAppliedACL(instance, applied)
}

func disableAndDisconnect(ctx context.Context, control *redisclient.Client, username string) error {
	if err := control.Do(ctx, "ACL", "SETUSER", username, "reset", "off", "-@all").Err(); err != nil {
		return PolicyUnconfirmed
	}
	if err := control.Do(ctx, "CLIENT", "KILL", "USER", username).Err(); err != nil {
		return PolicyUnconfirmed
	}
	return nil
}

func removeVerifier(ctx context.Context, record *models.Database, username string) error {
	if !publicUsername(username) {
		return InvalidCredential
	}
	instance, err := Instance(ctx, record)
	if err != nil {
		return err
	}
	lock, err := lockOwnedFile(stateLockPath(record.ID), false)
	if err != nil {
		return Unavailable
	}
	defer lock.Close()
	document, err := readACLDocument(instance)
	if err != nil {
		return err
	}
	applied, err := readAppliedACL(instance)
	if err != nil {
		return err
	}
	control, err := privateClient(ctx, record, instance, systemProfile)
	if err != nil {
		return PolicyUnconfirmed
	}
	defer control.Close()
	actual, err := actualUser(ctx, control, username)
	if err != nil {
		return err
	}
	if _, exists := document.Users[username]; exists {
		delete(document.Users, username)
		if err := writeACLDocument(instance, document); err != nil {
			return err
		}
	}
	if actual != nil {
		if err := disableAndDisconnect(ctx, control, username); err != nil {
			return err
		}
		if err := control.Do(ctx, "ACL", "LOAD").Err(); err != nil {
			return PolicyUnconfirmed
		}
		actual, err = actualUser(ctx, control, username)
		if err != nil || actual != nil {
			return PolicyUnconfirmed
		}
	}
	if _, exists := applied.Users[username]; exists {
		delete(applied.Users, username)
		return writeAppliedACL(instance, applied)
	}
	return nil
}
