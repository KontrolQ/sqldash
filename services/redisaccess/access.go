package redisaccess

import (
	"context"
	"net"
	"net/netip"
	"time"

	"sqldash/config"
	"sqldash/database"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/redisgateway"
	"sqldash/redishost"
	"sqldash/services/access"
)

func Authorize(ctx context.Context, username string, peer netip.Addr) (*redisgateway.Binding, error) {
	if !config.Redis.Enabled || !models.PublicRedisUsername(username) || !peer.IsValid() {
		return nil, Refused
	}
	var credential models.RedisConnection
	if err := database.DB.WithContext(ctx).Where("username = ?", username).First(&credential).Error; err != nil {
		return nil, Refused
	}
	record, instance, err := validateCredential(ctx, &credential, peer)
	if err != nil {
		return nil, err
	}
	snapshot := &authority{AccountID: credential.AccountID, InstanceIdentity: instance.Identity, InstanceGeneration: instance.Generation, RunID: instance.RunID, Scope: credential.Scope, Subscribe: credential.Subscribe, Publish: credential.Publish, Peer: peer.Unmap()}
	return &redisgateway.Binding{Username: credential.Username, DatabaseID: record.ID, CredentialID: credential.ID, Generation: credential.Generation, State: snapshot}, nil
}

func Validate(ctx context.Context, binding *redisgateway.Binding, peer netip.Addr) error {
	if binding == nil || !config.Redis.Enabled || !models.PublicRedisUsername(binding.Username) {
		return Refused
	}
	snapshot, valid := binding.State.(*authority)
	if !valid || snapshot == nil || !peer.IsValid() || snapshot.Peer != peer.Unmap() {
		return Refused
	}
	var credential models.RedisConnection
	if err := database.DB.WithContext(ctx).First(&credential, binding.CredentialID).Error; err != nil {
		return Refused
	}
	if credential.DatabaseID != binding.DatabaseID || credential.AccountID != snapshot.AccountID || credential.Username != binding.Username || credential.Generation != binding.Generation || credential.InstanceIdentity != snapshot.InstanceIdentity || credential.Scope != snapshot.Scope || credential.Subscribe != snapshot.Subscribe || credential.Publish != snapshot.Publish {
		return Refused
	}
	_, instance, err := validateCredential(ctx, &credential, peer)
	if err != nil {
		return err
	}
	if instance.Identity != snapshot.InstanceIdentity || instance.Generation != snapshot.InstanceGeneration || instance.RunID != snapshot.RunID {
		return Refused
	}
	return nil
}

func validateCredential(ctx context.Context, credential *models.RedisConnection, peer netip.Addr) (*models.Database, *models.RedisInstance, error) {
	if !credential.Usable(time.Now()) || !peer.IsValid() {
		return nil, nil, Refused
	}
	store := database.DB.WithContext(ctx)
	var record models.Database
	if err := store.First(&record, credential.DatabaseID).Error; err != nil || record.Kind() != models.EngineRedis || !record.Available() || record.ClusterID != credential.InstanceIdentity {
		return nil, nil, Refused
	}
	instance, err := redishost.Instance(ctx, &record)
	if err != nil || instance.State != models.DatabaseReady || instance.Identity != credential.InstanceIdentity || instance.RunID == "" {
		return nil, nil, Refused
	}
	issuer, err := access.LoadWith(store, credential.AccountID)
	if err != nil || !issuer.Can(permissions.RedisCredentialsManage) || !issuer.Can(permissions.RedisDataRead) {
		return nil, nil, Refused
	}
	if credential.Scope == models.ScopeReadWrite && !issuer.Can(permissions.RedisDataWrite) || credential.PrimaryConnection && credential.Scope != models.ScopeReadWrite || credential.Subscribe && !issuer.Can(permissions.RedisSubscribe) || credential.Publish && !issuer.Can(permissions.RedisPublish) {
		return nil, nil, Refused
	}
	var visible int64
	if err := access.Scope(store.Model(&models.Database{}), issuer).Where("id = ?", record.ID).Count(&visible).Error; err != nil || visible != 1 {
		return nil, nil, Refused
	}
	var rules []models.AccessRule
	if err := store.Where("database_name = ?", record.Name).Find(&rules).Error; err != nil {
		return nil, nil, Refused
	}
	allowed := len(rules) == 0
	for _, rule := range rules {
		prefix, err := netip.ParsePrefix(rule.Network)
		if err != nil {
			return nil, nil, Refused
		}
		allowed = allowed || prefix.Contains(peer.Unmap())
	}
	if !allowed {
		return nil, nil, Refused
	}
	return &record, instance, nil
}

func Dial(ctx context.Context, binding *redisgateway.Binding) (net.Conn, error) {
	if binding == nil {
		return nil, Refused
	}
	snapshot, valid := binding.State.(*authority)
	if !valid || snapshot == nil || Validate(ctx, binding, snapshot.Peer) != nil {
		return nil, Refused
	}
	var record models.Database
	if err := database.DB.WithContext(ctx).First(&record, binding.DatabaseID).Error; err != nil {
		return nil, Refused
	}
	return redishost.Dial(ctx, &record)
}
