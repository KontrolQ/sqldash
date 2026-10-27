package redishost

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strings"

	"sqldash/models"
)

func validUsername(username string) bool {
	return models.ValidRedisUsername(username)
}

func passwordHash(password []byte) string {
	digest := sha256.Sum256(password)
	return hex.EncodeToString(digest[:])
}

func validPasswordHash(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}

func profileRules(profile Profile, subscribe, publish bool) (string, error) {
	base := "reset on ~* resetchannels +ping +hello +echo +quit +select +client|id +client|setname +client|setinfo"
	switch profile {
	case systemProfile:
		return "reset on ~* &* +@all", nil
	case ReadOnly, ReadWrite:
		base += " +@read +@transaction +eval_ro +evalsha_ro +fcall_ro +script|exists +script|load"
		if profile == ReadWrite {
			base += " +@write +@scripting"
		}
		base += " -@admin -@dangerous +keys -migrate -restore-asking -script|debug -script|kill -function|kill"
		for _, command := range vectorReadCommands {
			base += " +" + command
		}
		if profile == ReadWrite {
			base += " +restore +flushdb +flushall"
			for _, command := range vectorWriteCommands {
				base += " +" + command
			}
		}
	case Inspect:
		base += " +info +dbsize +time +memory|stats +memory|usage +slowlog|get +slowlog|len +client|list +command|info +command|docs +command|list"
	case Monitor:
		base += " +monitor"
	case Subscribe:
		subscribe = true
	case Publish:
		publish = true
	default:
		return "", InvalidCredential
	}
	if subscribe || publish {
		base += " &*"
	}
	if subscribe {
		base += " +subscribe +psubscribe +unsubscribe +punsubscribe"
	}
	if publish {
		base += " +publish"
	}
	return base, nil
}

func readACLDocument(instance *models.RedisInstance) (*aclDocument, error) {
	contents, err := readOwnedFile(aclStatePath(instance.DatabaseID))
	if err != nil {
		return nil, err
	}
	var document aclDocument
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&document) != nil || document.DatabaseID != instance.DatabaseID || document.Identity != instance.Identity || len(document.Users) > 1032 {
		return nil, IdentityMismatch
	}
	for _, profile := range systemProfiles {
		username := systemUsername(instance.DatabaseID, profile)
		user, present := document.Users[username]
		password, err := systemPassword(instance, profile, false)
		if err != nil {
			return nil, err
		}
		matches := present && user.Profile == profile && user.Hash == passwordHash(password)
		clear(password)
		if !matches {
			return nil, IdentityMismatch
		}
	}
	for username, user := range document.Users {
		if !validUsername(username) || !validPasswordHash(user.Hash) {
			return nil, InvalidCredential
		}
		if _, err := profileRules(user.Profile, user.Subscribe, user.Publish); err != nil {
			return nil, err
		}
	}
	return &document, nil
}

func writeACLDocument(instance *models.RedisInstance, document *aclDocument) error {
	encoded, err := json.Marshal(document)
	if err != nil || len(encoded) > maximumManifestBytes {
		return InvalidCredential
	}
	var contents strings.Builder
	contents.WriteString("user default reset off\n")
	usernames := make([]string, 0, len(document.Users))
	for username := range document.Users {
		usernames = append(usernames, username)
	}
	sort.Strings(usernames)
	for _, username := range usernames {
		user := document.Users[username]
		rules, err := profileRules(user.Profile, user.Subscribe, user.Publish)
		if err != nil || !validUsername(username) || !validPasswordHash(user.Hash) {
			return InvalidCredential
		}
		contents.WriteString("user ")
		contents.WriteString(username)
		contents.WriteByte(' ')
		contents.WriteString(rules)
		contents.WriteString(" #")
		contents.WriteString(user.Hash)
		contents.WriteByte('\n')
	}
	if err := ownedFile(aclStatePath(instance.DatabaseID), encoded, 0o600, 0); err != nil {
		return err
	}
	_, groupID, err := instanceCredentials(instance.DatabaseID)
	if err != nil {
		return err
	}
	return ownedFile(aclPath(instance.DatabaseID), []byte(contents.String()), 0o640, groupID)
}

func prepareACL(instance *models.RedisInstance) error {
	document, err := readACLDocument(instance)
	if errors.Is(err, os.ErrNotExist) {
		if _, err := os.Lstat(configurationPath(instance.DatabaseID)); !errors.Is(err, os.ErrNotExist) {
			return IdentityMismatch
		}
		entries, err := os.ReadDir(dataDirectory(instance.DatabaseID))
		if err != nil || len(entries) != 0 {
			return StorageUnowned
		}
		document = &aclDocument{DatabaseID: instance.DatabaseID, Identity: instance.Identity, Users: make(map[string]aclUser)}
		referenceKey, err := systemPassword(instance, referenceSecret, true)
		clear(referenceKey)
		if err != nil {
			return err
		}
		for _, profile := range systemProfiles {
			password, err := systemPassword(instance, profile, true)
			if err != nil {
				return err
			}
			document.Users[systemUsername(instance.DatabaseID, profile)] = aclUser{Hash: passwordHash(password), Profile: profile}
			clear(password)
		}
	} else if err != nil {
		return err
	}
	return writeACLDocument(instance, document)
}

func publicUsername(username string) bool {
	return models.PublicRedisUsername(username)
}

func SetUser(ctx context.Context, record *models.Database, username, password string, profile Profile, subscribe, publish bool) error {
	if len(password) < 32 || len(password) > 1024 {
		return InvalidCredential
	}
	return SetVerifier(ctx, record, username, passwordHash([]byte(password)), profile, subscribe, publish)
}

func RemoveUser(ctx context.Context, record *models.Database, username string) error {
	return removeVerifier(ctx, record, username)
}
